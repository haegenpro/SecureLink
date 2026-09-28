# Architecture

This document explains the *why* behind SecureLink's design. For *what* the
system does, see [README.md](README.md).

## Why S3 for object storage

File bytes are large, variable-sized, and don't need relational structure —
S3 is the natural fit, and offloading storage to it means the API process
never has to hold file bytes in memory or on local disk, and never becomes a
throughput bottleneck for large uploads/downloads.

## Why presigned URLs instead of proxying uploads through the API

If the API accepted file bytes directly, every byte would flow
client → API → S3, doubling bandwidth and making the API's memory/CPU a
function of concurrent transfer size. Presigned URLs let the client talk to
S3 directly: the API's job is reduced to *authorizing* a transfer (by
minting a time-limited, single-purpose credential), which is a fast,
constant-size operation regardless of file size. This is also why
`GET /files/:id/download` never streams the file itself — it returns a URL.

The trade-off: the API can't inspect or reject file *content* before it
lands in S3 (only metadata, before the URL is issued). That's acceptable
here since content inspection is explicitly out of scope.

## Why processing is asynchronous

Nothing about "confirm this upload happened and record its status" needs to
block the client's HTTP request — and if it did, a slow or failing
processing step would directly translate into a slow or failing API,
coupling two independently-scaling concerns. Decoupling them via a queue
means the API's request latency is independent of processing latency, and a
processing backlog degrades gracefully (messages queue up) instead of
cascading into API timeouts.

## Why SQS

S3 can publish object-created events directly to an SQS queue. SQS gives us,
for free, exactly the properties a background job needs:

- **Durability**: a message survives even if no worker is running when it's
  published.
- **At-least-once delivery** with a **visibility timeout**: a message a
  worker fails to acknowledge automatically becomes available for another
  attempt, without any custom retry bookkeeping.
- **Decoupling**: S3 and the worker never need to know about each other
  directly.

The cost is that "at-least-once" means duplicates are possible — which is
why idempotency (below) is not optional.

## How duplicate messages are handled

SQS's at-least-once guarantee means the same S3 event can be delivered more
than once (e.g. a worker crashes after processing but before deleting the
message, or a retry fires while the original is still in flight). The
worker (`internal/worker/worker.go`, `handleEvent`) makes this safe with an
explicit status check:

```text
if file.status == processed:
    acknowledge (delete) the message
    return without reprocessing
otherwise:
    mark processing
    run the processor
    mark processed / failed
    acknowledge only after a terminal state is durably written
```

This is a check-then-act sequence, not a database-enforced compare-and-swap,
so it is *not* safe against two workers processing the same message
concurrently at the exact same instant (a genuine race would need something
like `UPDATE ... WHERE status != 'processed'` with a rows-affected check).
For a single-worker, one-day-project scale this is an accepted trade-off,
called out explicitly rather than silently assumed away.

## What happens when the worker crashes

- **Before marking `processing`**: the message is still in the queue,
  invisible until the visibility timeout expires, then redelivered. No state
  was written, so reprocessing from scratch is safe.
- **After marking `processing` but before finishing**: the file is stuck in
  `processing` until the message is redelivered (visibility timeout) and a
  worker picks it up again. Because the idempotency check only special-cases
  `processed`, a file in `processing` *will* be reprocessed on redelivery —
  which is the intended, safe behavior for this project's `NoopProcessor`,
  since redoing a no-op is harmless. A processor with real side effects would
  need to make its own work idempotent, or the state machine would need a
  distinct "processing" claim with a heartbeat/lease, which this project does
  not implement.
- **After marking `processed`/`failed` but before deleting the message**:
  the message is redelivered, but the idempotency check (`processed`) short-
  circuits it immediately (a `failed` file is *not* short-circuited today —
  see "what this system does not guarantee").

## What this system deliberately does not guarantee

- **Exactly-once processing.** Only duplicate-after-`processed` is
  guarded against. A file left in `failed` will be retried if its message is
  redelivered (there is no terminal "don't retry `failed`" check) — this was
  a deliberate scope cut rather than an oversight, since retrying a failed,
  no-op processing step is harmless in this project.
- **Concurrent-worker safety.** The status check-then-act is not atomic
  across multiple worker processes (see above).
- **Content integrity/scanning.** The worker confirms an event arrived; it
  does not scan, validate, or transform file contents.
- **Multi-node metadata storage.** SQLite serializes writers via file
  locking; it works for the API and worker sharing one file (including
  across containers via a shared volume, as in `docker-compose.yml`), but
  does not scale to multiple hosts without a real database server.
- **Delivery ordering.** SQS standard queues (used here) do not guarantee
  order; this project's state machine doesn't depend on ordering, but it's
  worth naming explicitly.
