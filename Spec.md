# SecureLink — Cloud-Native Secure File Transfer Service

**Status:** Implementation specification  
**Target:** One-day portfolio project  
**Primary goal:** Demonstrate practical Go backend, AWS, asynchronous processing, Docker, Terraform, CI, Linux/Unix, and security fundamentals for a software-engineering-oriented CV.

## 1. Project Overview

SecureLink is a deliberately small secure file-transfer backend.

A client requests a short-lived upload URL from a Go API, uploads the file directly to Amazon S3, and an asynchronous processing pipeline is triggered through Amazon SQS. A separate Go worker consumes the job, validates/processes the file metadata, records the processing state, and acknowledges the queue message.

The project should remain small enough to understand completely. It is **not** intended to become a Dropbox clone, production SaaS, or Kubernetes exercise.

### Core architecture

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
                    │      download  │
                    │ GET /health     │
                    └────────┬────────┘
                             │
                    presigned S3 URL
                             │
                             ▼
                    ┌─────────────────┐
                    │    AWS S3       │
                    │ private bucket   │
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

## 2. CV-Relevance

The project should produce genuine evidence for:

- **Go:** API server and background worker.
- **AWS:** S3 and SQS; IAM where practical.
- **Cloud-native architecture:** direct object storage upload + asynchronous worker.
- **Distributed-systems fundamentals:** retries, duplicate delivery, idempotency, failure isolation.
- **Docker:** reproducible local/container execution.
- **Terraform:** infrastructure as code.
- **GitHub Actions:** automated test/lint/build pipeline.
- **Linux/Unix:** development and operational commands.
- **Security:** private S3 bucket, least-privilege IAM, short-lived presigned URLs.
- **Technical communication:** architecture diagram, README, trade-offs, failure scenarios.

The user's current CV already demonstrates Python, SQL, Java, C/C++, Go, TypeScript, React/Next.js, REST APIs and substantial academic/ML experience; SecureLink should therefore add **credible cloud/backend/DevOps evidence**, not repeat existing strengths. fileciteturn0file1L12-L16

## 3. Scope

### Must Have

1. Go REST API.
2. `POST /files` to create file metadata and a short-lived S3 upload URL.
3. `GET /files/:id` to retrieve file metadata/status.
4. `GET /files/:id/download` to create a short-lived download URL.
5. `GET /health`.
6. Private S3 bucket.
7. S3 → SQS event flow.
8. Go worker consuming SQS.
9. Idempotent processing behavior.
10. Dockerfile.
11. Terraform for core AWS infrastructure.
12. GitHub Actions CI running Go tests, `go vet`, and Docker build.
13. README with architecture and local setup.
14. Tests for important API and worker behavior.

### Nice to Have

Only implement these after the mandatory path works:

- PostgreSQL or another persistent metadata store.
- ECR image publishing.
- Automated AWS deployment.
- Dead-letter queue.
- Structured JSON logging.
- Prometheus-style metrics.
- Virus-scanning integration.
- Authentication/RBAC.
- Kubernetes/ECS deployment.

### Explicitly Out of Scope

Do **not** build:

- A frontend.
- User registration/login.
- A full authentication system.
- A dashboard.
- Multi-tenant billing.
- Kubernetes unless the core project is already complete.
- Complex file transformation.
- A production-grade malware scanner.
- A large microservice ecosystem.

## 4. Functional Requirements

### 4.1 Create Upload

`POST /files`

The API should:

1. Generate a unique file ID.
2. Accept basic file metadata such as filename and content type.
3. Create an S3 object key.
4. Persist or temporarily retain metadata.
5. Return a short-lived presigned PUT URL.

Example response:

```json
{
  "file_id": "abc123",
  "upload_url": "https://...",
  "expires_in": 300
}
```

The URL should expire quickly (target: 5 minutes).

### 4.2 File Status

`GET /files/:id`

Example:

```json
{
  "id": "abc123",
  "status": "processed",
  "filename": "report.pdf",
  "content_type": "application/pdf",
  "size": 183920,
  "created_at": "2026-09-27T10:00:00Z"
}
```

Suggested statuses:

```text
pending
processing
processed
failed
```

### 4.3 Download

`GET /files/:id/download`

Return a short-lived presigned GET URL for a valid file.

Do not proxy the file contents through the API.

### 4.4 Health

`GET /health`

Return HTTP 200 when the API process is alive.

Example:

```json
{
  "status": "ok"
}
```

## 5. Asynchronous Processing

The API must not perform potentially slow file processing synchronously.

Expected flow:

```text
S3 upload
  ↓
S3 event
  ↓
SQS message
  ↓
worker receives message
  ↓
worker validates event
  ↓
worker processes metadata
  ↓
worker records status
  ↓
worker acknowledges message
```

### Idempotency

SQS-style delivery must be treated as potentially duplicated.

The worker must not corrupt state if the same event is delivered more than once.

At minimum:

```text
if file is already processed:
    acknowledge message
    exit successfully

otherwise:
    mark processing
    process
    mark processed
    acknowledge message
```

The implementation should make the idempotency mechanism explicit in code and documentation.

## 6. Error Handling

The system should distinguish:

- Invalid client input → `4xx`.
- Missing file → `404`.
- Internal application failure → `5xx`.
- Temporary worker failure → allow queue retry.
- Permanent processing failure → mark the file as `failed`.

Do not acknowledge an SQS message before successful processing unless the failure is intentionally non-retryable.

If a dead-letter queue is implemented, document its purpose and retry policy.

## 7. Security Requirements

Minimum security baseline:

- S3 bucket must not be publicly readable.
- Do not hard-code AWS credentials.
- Use environment variables / AWS credential provider mechanisms locally.
- Use IAM permissions with least privilege.
- Presigned URLs must be short-lived.
- Do not expose secret credentials in logs.
- Validate filenames and metadata.
- Do not construct unsafe filesystem paths from user-controlled filenames.
- Keep secrets out of Git.
- Provide `.env.example`, never a real `.env` with credentials.
- Terraform should avoid committing sensitive values.

Security claims in the README must reflect what is actually implemented.

## 8. Suggested Repository Structure

```text
securelink/
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── internal/
│   ├── api/
│   ├── domain/
│   ├── storage/
│   ├── queue/
│   └── worker/
├── terraform/
│   ├── main.tf
│   ├── variables.tf
│   ├── outputs.tf
│   ├── iam.tf
│   └── versions.tf
├── .github/
│   └── workflows/
│       └── ci.yml
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
├── README.md
├── ARCHITECTURE.md
└── .gitignore
```

The exact package layout may differ if Claude Code identifies a materially simpler idiomatic Go structure.

## 9. Local Development

The project must be runnable locally without AWS wherever reasonably possible.

Preferred approach:

- Use environment variables for AWS configuration.
- Use LocalStack only if it materially reduces development friction.
- Keep a documented path for testing against real AWS.
- Docker should build successfully.
- `make test` (or equivalent) should execute the complete test suite.

Suggested commands:

```bash
make test
make vet
make build
make docker
```

## 10. Testing Requirements

At minimum:

### API tests

- Health endpoint returns 200.
- Invalid create-file request returns 4xx.
- Unknown file ID returns 404.
- Successful create-file request returns an ID and upload URL.

### Worker tests

- Valid event is processed.
- Duplicate event is safe/idempotent.
- Processing failure does not falsely acknowledge a retryable message.
- Malformed event is handled safely.

### Infrastructure/CI

CI should run:

```text
go test ./...
go vet ./...
docker build .
```

Do not add a linting dependency merely for the sake of a CV keyword unless it provides useful signal.

## 11. Terraform Requirements

Terraform should provision, at minimum:

- S3 bucket.
- SQS queue.
- S3 → SQS notification/event wiring.
- IAM resources/policies needed by the API/worker.
- ECR repository only if image publishing is implemented.

Terraform should use variables for environment-specific values.

Avoid unnecessary resources.

## 12. CI Requirements

GitHub Actions should run on pull requests and pushes.

Minimum pipeline:

```text
checkout
  ↓
setup Go
  ↓
go test ./...
  ↓
go vet ./...
  ↓
docker build
```

The pipeline should fail when tests fail.

Deployment is optional and should never block completion of the core project.

## 13. Observability

Minimum:

- Useful structured/logged messages for API startup, worker receipt, processing success, and processing failure.
- Never log secrets or presigned URLs in full.
- Include file IDs/job IDs where useful for tracing.

Optional:

- JSON logging.
- Metrics.
- Request IDs.

## 14. Documentation Requirements

`README.md` must include:

1. Project purpose.
2. Architecture diagram.
3. Technology stack.
4. Prerequisites.
5. Local setup.
6. Environment variables.
7. Running tests.
8. Running API and worker.
9. AWS deployment instructions, if implemented.
10. Security considerations.
11. Failure/retry behavior.
12. Design trade-offs.
13. Future improvements.

`ARCHITECTURE.md` should explain:

- Why S3 is used for object storage.
- Why uploads use presigned URLs.
- Why processing is asynchronous.
- Why SQS is used.
- How duplicate messages are handled.
- What happens when the worker crashes.
- What the system deliberately does not guarantee.

## 15. Definition of Done

The project is complete when all of the following are true:

- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] API starts successfully.
- [ ] `POST /files` returns a usable upload URL.
- [ ] A real file can be uploaded to S3.
- [ ] S3 produces the expected queue event.
- [ ] Worker consumes and processes the event.
- [ ] Duplicate processing is safe.
- [ ] File status can be retrieved.
- [ ] Download URL works.
- [ ] Docker image builds.
- [ ] Terraform validates/plans successfully.
- [ ] GitHub Actions CI passes.
- [ ] README explains the actual implementation.
- [ ] No credentials or secrets are committed.
- [ ] Git history contains no accidental secrets.

## 16. One-Day Execution Priority

If time runs out, use this priority order:

1. Go API.
2. S3 upload/download flow.
3. SQS event flow.
4. Worker.
5. Idempotency and tests.
6. Docker.
7. Terraform.
8. GitHub Actions.
9. Documentation polish.
10. Optional deployment.

A working end-to-end system is more valuable than an incomplete collection of cloud services.

## 17. CV Evidence

Only after the implementation is genuinely completed, tested, and understood, a CV entry can be based on evidence such as:

**SecureLink — Cloud-Native Secure File Transfer Service**  
*Go · AWS S3/SQS · Docker · Terraform · GitHub Actions*

- Built a containerized Go service using short-lived S3 presigned URLs for secure file-transfer workflows and asynchronous SQS-based processing.
- Designed retry-safe, idempotent background processing and provisioned core AWS infrastructure with Terraform, with automated Go tests and Docker builds through GitHub Actions.

Do not use these bullets verbatim if the final implementation does not support every claim.
