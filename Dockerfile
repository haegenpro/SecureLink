# Multi-stage build: compile static binaries, then ship them in a minimal
# runtime image with no Go toolchain or shell.
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

# A pre-owned /data directory so the SQLite file can be created under the
# nonroot user below, even when /data is a fresh, root-owned named volume:
# Docker seeds a first-mounted named volume from the image's existing
# directory contents and ownership.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot AS base
COPY --from=builder --chown=nonroot:nonroot /out/data /data

FROM base AS api
COPY --from=builder /out/api /usr/local/bin/api
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/api"]

FROM base AS worker
COPY --from=builder /out/worker /usr/local/bin/worker
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/worker"]
