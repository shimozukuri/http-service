# http-service

`http-service` is a small URL-shortening HTTP API written in Go. It stores URL-to-alias mappings in SQLite, exposes public redirects and health checks, and protects write operations with HTTP Basic Authentication.

## Features

- Create short aliases for valid URLs.
- Generate a six-character alias when none is supplied.
- Redirect public requests from an alias to the stored URL.
- Delete existing aliases.
- Persist mappings in a local SQLite database created on startup.
- Protect create and delete operations with HTTP Basic Authentication.
- Emit structured request logs, request IDs, and panic recovery middleware.
- Provide unit and end-to-end functional tests.

## Tech stack

- Go 1.26
- [chi v5](https://github.com/go-chi/chi) for routing and middleware
- `log/slog` for logging
- [cleanenv](https://github.com/ilyakaznacheev/cleanenv) for YAML and environment-based configuration
- SQLite through `github.com/mattn/go-sqlite3`
- `go-playground/validator` for request validation
- Testify, Mockery-generated mocks, and HTTPExpect for testing
- GitHub Actions for continuous integration

## Project structure

```text
.
|-- cmd/http-service/                  # Application entry point and route setup
|-- config/                            # Runtime configuration used by CI
|-- internal/
|   |-- config/                        # Configuration model and loader
|   |-- http-server/
|   |   |-- handlers/                  # Save, delete, and redirect handlers
|   |   `-- middleware/logger/         # Request logging middleware
|   |-- lib/                           # API responses, logging helpers, and alias generation
|   `-- storage/
|       `-- sqlite/                    # SQLite storage implementation and schema setup
|-- tests/                             # Functional tests (functional build tag)
`-- .github/workflows/test.yaml        # Unit and functional test workflow
```

## Getting started

### Prerequisites

- Go 1.26 or later
- CGO enabled and a C compiler available, as required by `go-sqlite3`

Clone the repository and download its dependencies:

```bash
git clone https://github.com/shimozukuri/http-service.git
cd http-service
go mod download
```

Create the directory that will contain the SQLite database:

```bash
mkdir -p storage
```

### Configuration

The application requires `CONFIG_PATH` to point to a configuration file. The repository includes `config/ci.yaml` with the settings used by the functional tests and GitHub Actions:

```yaml
env: "ci"
storage_path: "./storage/storage.db"
http_server:
  address: "localhost:8082"
  timeout: 4s
  idle_timeout: 60s
  user: "myuser"
  password: "mypass"
```

| Setting | Required | Default | Description |
|---|---:|---|---|
| `env` | No | `local` | Logging mode. The application implements `local`, `dev`, and `prod`. |
| `storage_path` | Yes | None | Path to the SQLite database file. Its parent directory must exist. |
| `http_server.address` | No | `localhost:8082` | HTTP listen address. |
| `http_server.timeout` | No | `4s` | HTTP read and write timeout. |
| `http_server.idle_timeout` | No | `60s` | HTTP idle timeout. |
| `http_server.user` | Yes | None | Basic Authentication username for write endpoints. |
| `http_server.password` | Yes | None | Basic Authentication password for write endpoints. |

Environment variables:

| Variable | Required | Description |
|---|---:|---|
| `CONFIG_PATH` | Yes | Path to the YAML configuration file. The process exits if it is unset or the file does not exist. |
| `HTTP_SERVER_PASSWORD` | No | Overrides `http_server.password` from the configuration file. |

The application currently implements logger modes for `local`, `dev`, and `prod`. Do not reuse or commit production credentials; supply production passwords through `HTTP_SERVER_PASSWORD` or another secret-management mechanism.

### Run locally

On Linux or macOS:

```bash
export CONFIG_PATH=./config/ci.yaml
go run ./cmd/http-service
```

On PowerShell:

```powershell
$env:CONFIG_PATH = ".\config\ci.yaml"
go run .\cmd\http-service
```

With the example configuration, the service listens at `http://localhost:8082` and creates the SQLite schema automatically.

## API

Create and delete routes require Basic Authentication. Redirect and health routes are public.

| Method | Path | Authentication | Success | Description |
|---|---|---|---:|---|
| `POST` | `/url` | Basic | `201 Created` | Save a URL with an optional alias. |
| `DELETE` | `/url/{alias}` | Basic | `200 OK` | Delete an alias. |
| `GET` | `/{alias}` | None | `302 Found` | Redirect to the stored URL. |
| `GET` | `/health` | None | `200 OK` | Readiness endpoint with an empty response body. |

JSON responses use a common status envelope:

```json
{
  "status": "OK",
  "alias": "docs"
}
```

Errors use `status: "ERROR"` and include an `error` field:

```json
{
  "status": "ERROR",
  "error": "not found"
}
```

### Create a short URL

Supply a custom alias:

```bash
curl -i -u myuser:mypass \
  -H "Content-Type: application/json" \
  -d '{"url":"https://go.dev/doc/","alias":"go-docs"}' \
  http://localhost:8082/url
```

Omit `alias` to generate a six-character alias:

```bash
curl -i -u myuser:mypass \
  -H "Content-Type: application/json" \
  -d '{"url":"https://go.dev/doc/"}' \
  http://localhost:8082/url
```

Relevant responses are `201 Created`, `400 Bad Request` for malformed or invalid JSON input, `401 Unauthorized` for missing or invalid credentials, `409 Conflict` for an existing alias, and `500 Internal Server Error` for storage failures.

### Follow a redirect

Inspect the `302 Found` response and `Location` header:

```bash
curl -i http://localhost:8082/go-docs
```

Use `-L` to follow the redirect:

```bash
curl -L http://localhost:8082/go-docs
```

The endpoint returns `404 Not Found` when the alias does not exist and `500 Internal Server Error` when storage lookup fails.

### Delete a short URL

```bash
curl -i -u myuser:mypass \
  -X DELETE \
  http://localhost:8082/url/go-docs
```

A successful deletion returns:

```json
{
  "status": "OK"
}
```

The endpoint returns `401 Unauthorized` for missing or invalid credentials, `404 Not Found` when the alias does not exist, and `500 Internal Server Error` for storage failures.

### Check service health

```bash
curl -i http://localhost:8082/health
```

## Tests

Run the unit tests, including the race detector, with the same command used by CI:

```bash
go test -v -race ./...
```

Functional tests use the `functional` build tag and expect a running service at `http://localhost:8082` with Basic Authentication credentials `myuser` / `mypass`. The included `config/ci.yaml` provides these settings.

Start the service in one terminal:

```bash
CONFIG_PATH=./config/ci.yaml go run ./cmd/http-service
```

Then run the functional suite in another terminal:

```bash
go test -tags=functional -v -count=1 ./tests/...
```

## CI/CD

The `Tests` GitHub Actions workflow runs for pull requests and pushes to `master`:

1. The `unit-tests` job checks out the repository, installs the Go version from `go.mod`, downloads dependencies, and runs `go test -v -race ./...`.
2. After unit tests pass, the `functional-tests` job builds `./cmd/http-service`, starts it with `CONFIG_PATH=./config/ci.yaml`, polls `/health` until it is ready, and runs the functional suite with the `functional` build tag.
3. Application logs are printed if the functional job fails.

The repository currently defines continuous integration only; no deployment workflow is configured.

## Roadmap

- Validate unsupported `env` values during configuration loading.
- Add graceful HTTP server shutdown and explicit storage cleanup.
- Introduce versioned database migrations and direct storage tests.
- Add container packaging and deployment automation.
- Consider rate limiting and stronger production credential management.
