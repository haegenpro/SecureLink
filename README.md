# SecureLink — Cloud-Native Secure File Transfer Service

A deliberately small secure file-transfer backend: a Go API issues short-lived
presigned S3 URLs so clients upload/download directly to/from S3, and a
separate Go worker processes S3 object-created events delivered through SQS,
idempotently.

This is a one-day portfolio project. It is not a Dropbox clone, has no
frontend, and has no user authentication — see [Scope](#scope) and
[Trade-offs](#design-trade-offs) below for what was deliberately left out.

## Architecture

```text
                    ┌─────────────────┐
                    │      Client     │
                    └────────┬────────┘
                             │ HTTPS / REST
                             ▼
                    ┌─────────────────┐
                    │   Go API Server │
                    │                 │
                    │ POST /files     │
                    │ GET  /files/:id │
                    │ GET /files/:id/ │
                    │      download   │
                    │ GET /health     │
                    └────────┬────────┘
                             │
                    presigned S3 URL
                             │
                             ▼
                    ┌─────────────────┐
                    │    AWS S3       │
                    │ private bucket  │
                    └────────┬────────┘
                             │ object event
                             ▼
                    ┌─────────────────┐
                    │    Amazon SQS   │
                    └────────┬────────┘
                             │ job
                             ▼
                    ┌─────────────────┐
                    │    Go Worker    │
                    │                 │
                    │ validate        │
                    │ process         │
                    │ update status   │
                    │ acknowledge     │
                    └─────────────────┘
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the rationale behind each piece.

## Technology stack

- **Go 1.25** — API server (`cmd/api`) and worker (`cmd/worker`), stdlib
  `net/http` (no web framework).
- **AWS SDK for Go v2** — S3 presigned URLs, SQS receive/delete.
- **SQLite** (`modernc.org/sqlite`, pure Go, no CGO) — file metadata store.
- **Docker** — multi-stage build producing two minimal distroless images
  (api, worker).
- **LocalStack** (via `docker-compose.yml`) — local S3/SQS emulation, so the
  whole pipeline runs without an AWS account.
- **Terraform** — S3 bucket, SQS queue, S3→SQS event notification, IAM.
- **GitHub Actions** — `go test`, `go vet`, Docker build, Terraform
  fmt/validate on every push/PR.

## Prerequisites

- Go 1.25+
- Docker and Docker Compose (for the containerized local stack)
- Terraform 1.5+ (only needed to validate/plan infrastructure)
- An AWS account (only needed for a real deployment; everything else runs
  against LocalStack or in-process fakes)

## Local setup

### Option A — plain Go, SQLite only (fastest, no S3/SQS)

```bash
go run ./cmd/api
```

`GET /files/:id` and `POST /files` will work; `upload_url`/`download_url`
generation requires a configured S3 endpint (LocalStack or real AWS) since
presigning needs valid AWS credentials and a reachable region/endpoint.

### Option B — full stack via Docker Compose (LocalStack + api + worker)

```bash
docker compose up -d --build
curl http://localhost:8080/health
```

`docker-compose.yml` starts LocalStack (S3 + SQS), waits for it to be
healthy, provisions the bucket/queue/notification wiring via
[`scripts/localstack-init.sh`](scripts/localstack-init.sh), then starts the
API and worker pointed at LocalStack.

Manual end-to-end check:

```bash
# 1. Ask the API for an upload URL
curl -s -X POST http://localhost:8080/files \
  -H 'Content-Type: application/json' \
  -d '{"filename":"report.pdf","content_type":"application/pdf","size":1024}'
# => {"file_id":"...","upload_url":"...","expires_in":300}

# 2. Upload straight to S3 (via LocalStack) using the returned upload_url
curl -X PUT "<upload_url>" -H 'Content-Type: application/pdf' --data-binary @report.pdf

# 3. LocalStack fires an S3 event -> SQS -> the worker consumes it.
#    Poll the file's status:
curl -s http://localhost:8080/files/<file_id>
# status should move pending -> processing -> processed

# 4. Get a download URL
curl -s http://localhost:8080/files/<file_id>/download
```

## Environment variables

| Variable                     | Default (local)                                          | Purpose                                             |
| ----------------------------- | --------------------------------------------------------- | ---------------------------------------------------- |
| `API_ADDR`                   | `:8080`                                                   | HTTP listen address                                   |
| `DB_PATH`                    | `./data/securelink.db`                                    | SQLite database file (shared by API and worker)       |
| `AWS_REGION`                 | `us-east-1`                                               | AWS region                                            |
| `AWS_ENDPOINT_URL`           | empty                                                     | Set to a LocalStack URL (e.g. `http://localhost:4566`) to target it instead of real AWS |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | unset                                     | Only used (as static credentials) when `AWS_ENDPOINT_URL` is set, for LocalStack |
| `S3_BUCKET`                  | `securelink-files`                                        | S3 bucket for uploaded files                          |
| `SQS_QUEUE_URL`              | empty                                                     | SQS queue URL the worker polls                        |
| `UPLOAD_URL_TTL_SECONDS`     | `300`                                                     | Presigned PUT URL lifetime                            |
| `DOWNLOAD_URL_TTL_SECONDS`   | `300`                                                     | Presigned GET URL lifetime                            |
| `SQS_WAIT_TIME_SECONDS`      | `10`                                                      | SQS long-poll wait time                               |
| `SQS_MAX_MESSAGES`           | `5`                                                       | Max messages per SQS receive call                     |

See [`.env.example`](.env.example) for a ready-to-copy file (never commit a
real `.env`).

Real AWS credentials are read via the standard AWS SDK credential chain
(environment variables, shared config/credentials files, or an IAM role) —
never hard-coded.

## Running tests

```bash
go test ./...      # or: make test
go vet ./...        # or: make vet
```

38 tests across 8 packages cover: filename/content-type validation, config
defaults/overrides, the SQLite repository, S3 event parsing, and — most
importantly — the API handlers and the worker's idempotent processing state
machine (valid event, duplicate event, malformed event, transient failure,
permanent failure). All of these run against in-memory fakes; none require
AWS credentials.

## Running the API and worker

```bash
make run-api      # go run ./cmd/api
make run-worker    # go run ./cmd/worker
```

Both read configuration from the environment (see table above) and expect a
reachable S3/SQS endpoint (LocalStack or real AWS) to fully function; the API
alone can still serve `/health` and `GET /files/:id` against SQLite without
one.

## AWS deployment (documented, not applied by this project)

```bash
cd terraform
terraform init
terraform plan    # requires real AWS credentials
terraform apply
```

Terraform provisions: a private S3 bucket, an SQS queue, the S3→SQS
notification wiring, and least-privilege IAM policies/roles for the API and
worker (the roles are not attached to any compute resource — this project
does not implement automated deployment; see
[Not verified](#not-verified-against-real-aws) below).

## Security considerations

- The S3 bucket is provisioned with **Block Public Access** enabled and
  bucket-owner-enforced ACLs (no public read/write is possible).
- The API never handles file bytes: it only issues short-lived presigned
  URLs (default 5 minutes) so clients transfer directly to/from S3.
- AWS credentials are never hard-coded; they come from environment variables
  or the SDK's default credential chain. `.env` is git-ignored;
  `.env.example` contains no real secrets.
- Uploaded filenames are validated (`internal/domain.ValidateFilename`) to
  reject path separators and `.`/`..`, so a malicious filename can never be
  used to construct an unsafe path or escape its namespaced object key
  (`<file_id>/<filename>`).
- Presigned URLs are never logged in full — only the file ID and TTL are
  logged (see `internal/api/handlers.go`).
- IAM policies (`terraform/iam.tf`) are scoped to the specific S3/SQS actions
  and resource ARNs each component needs, not `*`.

## Failure and retry behavior

The worker treats SQS delivery as at-least-once and potentially duplicated:

- **Idempotency**: before processing, the worker checks the file's current
  status. If it is already `processed`, the worker acknowledges (deletes)
  the message without reprocessing.
- **Transient failure**: if the (simulated) processing step returns an
  ordinary error, the message is **not** deleted, so SQS redelivers it after
  the visibility timeout. The file is left in `processing` status.
- **Permanent failure**: if processing returns a `worker.PermanentError`,
  the file is marked `failed` and the message is acknowledged (deleted) —
  it will never succeed on retry, so it isn't left to loop forever.
- **Malformed messages**: a message body that isn't a valid S3 event
  notification is logged and deleted as a poison message (this project has
  no dead-letter queue in front of the main queue; see Trade-offs).

See [ARCHITECTURE.md](ARCHITECTURE.md) for more detail, including what
happens if the worker crashes mid-processing.

## Design trade-offs

- **SQLite instead of Postgres**: simpler for a one-day, single-node
  project; the `storage.FileRepository` interface makes swapping it for
  Postgres straightforward later (an explicit Nice-to-Have in the spec).
- **No dead-letter queue**: malformed messages are deleted rather than
  parked for inspection, trading observability of bad input for simplicity.
- **No real content processing**: `worker.NoopProcessor` never inspects
  file contents (no virus scanning, no transformation) — deliberately, per
  scope. It exists to demonstrate the pipeline's shape honestly.
- **No authentication**: anyone who can reach the API can create file
  records and receive presigned URLs. This mirrors many "share a link"
  services' MVP stage but would need auth before being internet-facing for
  real use.
- **Shared SQLite file between API and worker**: works because SQLite
  serializes writers via file locking, at the cost of not scaling past a
  single node without migrating to a real database.

## Future improvements

- Swap SQLite for PostgreSQL for multi-node deployments.
- Add a dead-letter queue with alerting for poison messages.
- Automated Terraform deployment + ECR image publishing in CI.
- Structured (JSON) logging and request IDs.
- Optional authentication layer in front of the API.

## Not verified against real AWS

This project was built and verified **locally only** (unit tests +
LocalStack via `docker-compose`). The following require a real AWS account
and were not executed as part of this build:

- `terraform apply` against a live AWS account.
- A real S3 upload triggering a real SQS event consumed by the worker.
- IAM policies attached to and exercised by real compute.

`terraform validate`/`fmt` were run locally and pass; `terraform plan`
against real AWS credentials is a documented manual follow-up.
