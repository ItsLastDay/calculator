# Calculator

A full-stack calculator modelled on the **Windows Standard calculator**
(without memory keys). A **React + TypeScript** frontend consumes a **Go REST
API**. The backend does all the arithmetic and owns the mathematical rules.
The frontend owns key handling, display and history. Both ship together in
one 15 MB, non-root container.

Operations: addition, subtraction, multiplication, division,
exponentiation (xʸ), square (x²), square root (√x), percent of (a % of b) and
sign change.

```
calculator/
├── backend/                       Go 1.23, standard library only
│   ├── api/openapi.yaml           the REST contract (OpenAPI 3.1), embedded and served
│   ├── cmd/server/                entry point: config, graceful shutdown, healthcheck
│   └── internal/
│       ├── calculator/            pure domain logic (no HTTP, no JSON)
│       └── api/                   HTTP boundary: handler, middleware, Problem Details
├── frontend/                      React 18, TypeScript (strict), Vite, Vitest
│   └── src/
│       ├── api/                   the only code that talks to the backend
│       ├── lib/engine.ts          pure key-by-key calculator state machine
│       ├── lib/keyboard.ts        keyboard shortcuts
│       ├── lib/format.ts          number formatting
│       ├── hooks/                 useCalculator (serial key queue), useKeyboard
│       ├── components/            Display, Keypad, History
│       └── docs/                  the /docs/ page (Swagger UI); HTML entry in frontend/docs/
├── docs/coverage/                 snapshot of coverage reports
├── Dockerfile                     multi-stage: test stages, build, distroless runtime
├── compose.yaml
├── Makefile
└── .github/workflows/ci.yml       lint, test (-race), coverage, image build, smoke test
```

---

## Quick start (Docker only)

Docker is the only requirement.

```bash
docker compose up --build
```

Open <http://localhost:8080> for the calculator, and
<http://localhost:8080/docs/> for the interactive API documentation.

Run the test suites, including linters and type checks, without installing Go
or Node:

```bash
docker build --target backend-test .
docker build --target frontend-test .
```

## Local development

Requirements: **Go 1.23+** and **Node 20+** (22 recommended).

**Backend** (terminal 1), listening on `:8080`:

```bash
cd backend
go run ./cmd/server
```

**Frontend** (terminal 2): Vite dev server on <http://localhost:5173> with hot
reload. It proxies `/api` to `localhost:8080`, so there is no CORS setup.

```bash
cd frontend
npm ci
npm run dev
```

With `make` available: `make dev-backend`, `make dev-frontend`, `make test`,
`make lint`, `make coverage`. Run `make help` for the full list.

### Configuration

The server reads environment variables and refuses to start on invalid
values.

| Variable           | Default  | Purpose                                                          |
| ------------------ | -------- | ---------------------------------------------------------------- |
| `ADDR`             | `:8080`  | Listen address                                                   |
| `STATIC_DIR`       | _(none)_ | Directory of built frontend files to serve; set in the container |
| `SHUTDOWN_TIMEOUT` | `10s`    | Grace period for in-flight requests on SIGTERM                   |

Frontend dev only: `API_PROXY_TARGET` (default `http://localhost:8080`).

---

## Using the calculator

The calculator works like the Windows one:
- **Left to right:** `2 + 3 × 4 =` gives **20**. Pressing an operator
  evaluates what is pending, and the line above the number shows the
  expression so far.
- **Repeat equals:** pressing `=` again repeats the last operation.
  `5 + =` uses 5 twice.
- **Percent of:** `%` is an operator between two numbers. `15 % 200 =` gives
  **30** (15% of 200) and shows `15% of 200 =`. The percent must not be
  negative ("Percentage cannot be negative"); the second number may be.
- **Errors:** the display shows the backend's explanation, for example
  "Square root of a negative number is not a real number." While one is
  shown, operators are disabled; type a number or press C/CE to continue.
- **History:** every result the backend calculates is recorded, with or
  without `=`: √x and x² steps (`√(√(16)) = 2`) and the intermediate results
  of chained operators (`2 + 3 × …` records `2 + 3 = 5`). Typing, `+/−`, C, CE
  and ⌫ calculate nothing, so they record nothing. Click an entry to bring its
  result back; the bin icon clears the history. History is kept in the
  browser tab's memory (the last 50 entries); the backend is stateless, so it
  is lost on reload.

| Key       | Keyboard    | Key        | Keyboard          |
| --------- | ----------- | ---------- | ----------------- |
| 0–9, `.`  | `0`–`9`, `.` or `,` | `=` | `Enter` or `=` |
| + − × ÷   | `+ - * /`   | `xʸ`       | `^`               |
| `x²`      | `q`         | `√x`       | `@`               |
| `%`       | `%`         | `+/−`      | `F9`              |
| `⌫`       | `Backspace` | `CE` / `C` | `Delete` / `Esc`  |

---

## REST API

**Interactive docs: <http://localhost:8080/docs/>.** The page uses Swagger UI,
self-hosted so the strict Content-Security-Policy stays in place. It renders
the contract the server publishes, and "Try it out" calls the live API. The
"API docs" link in the calculator header leads there.

The contract is [`backend/api/openapi.yaml`](backend/api/openapi.yaml). It is
embedded in the binary and served at `/api/v1/openapi.yaml`, so the published
document is exactly the checked-in one. A test also checks that every route
and every error code is documented.

| Method | Path                   | Success                    | Purpose                    |
| ------ | ---------------------- | -------------------------- | -------------------------- |
| `POST` | `/api/v1/calculations` | `200` `{"result": number}` | Evaluate a calculation     |
| `GET`  | `/api/v1/openapi.yaml` | `200` `application/yaml`   | The contract, for the docs |
| `GET`  | `/healthz`             | `204` (no body)            | Liveness probe             |

Request body: `{"operation": string, "operands": number[]}`.

| `operation`  | Operands | Meaning        | Example → `result` |
| ------------ | -------- | -------------- | ------------------ |
| `add`        | `[a, b]` | a + b          | `[2, 3]` → `5`     |
| `subtract`   | `[a, b]` | a − b          | `[2, 3]` → `-1`    |
| `multiply`   | `[a, b]` | a × b          | `[2.5, 4]` → `10`  |
| `divide`     | `[a, b]` | a ÷ b          | `[7, 2]` → `3.5`   |
| `power`      | `[a, b]` | a raised to b  | `[2, 10]` → `1024` |
| `sqrt`       | `[a]`    | √a             | `[16]` → `4`       |
| `percentage` | `[a, b]` | a percent of b; `a` must not be negative | `[15, 200]` → `30` |

### Examples

```bash
curl -i -X POST http://localhost:8080/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"operation": "add", "operands": [2, 3]}'
```

```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Request-Id: 4302447254fea16d

{"result":5}
```

```bash
curl -X POST http://localhost:8080/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"operation": "sqrt", "operands": [16]}'
# {"result":4}
```

### Errors

Errors are based on
[RFC 9457 Problem Details](https://www.rfc-editor.org/rfc/rfc9457)
(`application/problem+json`), with two members:

- `code`: a stable, machine-readable reason to branch on.
- `detail`: a sentence that is safe to show to users; the calculator
  displays it as is.

404, 405 and 500 apply to every path, so they are documented once at the top
of the spec rather than under each operation.

```bash
curl -i -X POST http://localhost:8080/api/v1/calculations \
  -H 'Content-Type: application/json' \
  -d '{"operation": "divide", "operands": [1, 0]}'
```

```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/problem+json

{"code":"DIVISION_BY_ZERO","detail":"Division by zero."}
```

| Status | `code`                   | When                                                                 |
| ------ | ------------------------ | -------------------------------------------------------------------- |
| 400    | `INVALID_REQUEST`        | Empty or malformed JSON, unknown fields, wrong types, missing fields |
| 400    | `UNSUPPORTED_OPERATION`  | `operation` is not one of the supported names                        |
| 400    | `INVALID_OPERAND_COUNT`  | Wrong number of operands for the operation                           |
| 400    | `INVALID_OPERAND`        | An operand is `null`                                                 |
| 404    | `NOT_FOUND`              | Unknown API path                                                     |
| 405    | `METHOD_NOT_ALLOWED`     | Wrong HTTP method (the `Allow` header lists the right one)           |
| 413    | `PAYLOAD_TOO_LARGE`      | Body over 64 KiB                                                     |
| 415    | `UNSUPPORTED_MEDIA_TYPE` | `Content-Type` is not JSON                                           |
| 422    | `DIVISION_BY_ZERO`       | `divide` by 0, or `power` with base 0 and a negative exponent        |
| 422    | `NEGATIVE_SQUARE_ROOT`   | `sqrt` of a negative number                                          |
| 422    | `NEGATIVE_PERCENTAGE`    | `percentage` with a negative percent, e.g. `[-10, 50]`               |
| 422    | `UNDEFINED_RESULT`       | Result is NaN, e.g. `power` of `-8` to `0.5`                         |
| 422    | `RESULT_OUT_OF_RANGE`    | Result overflows a 64-bit float, e.g. `1e308 × 10`                   |
| 500    | `INTERNAL_ERROR`         | Unexpected failure; details are logged, not returned                 |

---

## Testing and coverage

```bash
cd backend && go test -race -cover ./...
cd frontend && npm test             # or: npm run coverage
```

To fuzz the calculator beyond its seed corpus:

```bash
cd backend && go test -run '^$' -fuzz=FuzzCalculate -fuzztime=60s ./internal/calculator
```

Latest results (snapshot in [`docs/coverage/`](docs/coverage)). Open
`backend.html` or `frontend/index.html` there for line-by-line detail.

| Layer    | Scope                 | Statements                                                     |
| -------- | --------------------- | -------------------------------------------------------------- |
| Backend  | `internal/calculator` | 100%                                                           |
| Backend  | `internal/api`        | 96.8%                                                          |
| Backend  | `cmd/server`          | 49.3% (process wiring in `main`; config and healthcheck are tested) |
| Frontend | all of `src/` (144 tests) | 99.8% statements, 98% branches                             |

What the tests cover:

- **Domain** (`calculator_test.go`, about 740 cases). Expected values are
  exact, either representable or a single correctly rounded operation, so
  results are compared with `==`.
  - Magnitudes: the whole float64 range, from the smallest subnormal (2^-1074)
    to `MaxFloat64`, including underflow to zero and overflow boundaries.
  - Precision: integer exactness up to 2^53 and round-to-even beyond it, and
    IEEE behaviour for decimals (`0.1 + 0.2`).
  - Every error at the extremes, `-0` normalisation for every operation, and
    arity checks for every operation.
  - Identities (`v + 0 = v`, `v ÷ v = 1`, `100% of v = v`, …) across 20
    magnitudes, and commutativity over 2,000 random pairs.
  - A fuzz test comparing `Calculate` with an independently written
    specification for any input, including NaN, ±Inf and ±0. A 60-second run
    checked about 11.7 million inputs.
  - A guard test fails if an operation is added without being covered.
- **HTTP boundary:**
  - status codes and exact error messages for about 20 malformed or invalid
    requests
  - content-type handling, the body-size limit, 404/405, health, static files,
    request IDs, panic recovery and security headers.

  Responses are checked member by member, so a redundant field fails a test.
  A test also ensures every problem code is documented in the contract.
- **Concurrency:** 200 simultaneous requests through a real server, run under
  the race detector. Each response must match its own request.
- **Calculator engine** (67 tests): Windows behaviour key by key, including
  entry editing, left-to-right evaluation, repeat-equals, unary-key
  expressions, percent-of, error lock-out and history recall.
- **UI:** the API client against a mocked `fetch`, keyboard mapping, and
  component tests that click keys and type like a user: friendly errors,
  disabled keys, history, the busy indicator, and Enter behaviour.
- **Key ordering:** a fake backend that ignores aborts settles responses by
  hand. It proves that:
  - keys typed during a request apply in order afterwards
  - C cancels the request and drops queued keys
  - late responses or errors from before C never win
  - unmount cleans up
  - timeouts release the queue.

  Removing the stale-response guard makes these tests fail.

Linting: `gofmt`, `go vet` and `golangci-lint` for Go (see
`backend/.golangci.yml`). TypeScript `strict`, ESLint (typescript-eslint
strict type-checked, react-hooks) and Prettier for the frontend. CI runs all
of them.

---

## Design decisions

**Clear boundaries.** Each layer talks to the next through one narrow,
typed interface:

| Layer                       | Knows about           | Exposes                                    |
| --------------------------- | --------------------- | ------------------------------------------ |
| `internal/calculator`       | numbers only          | `Calculate(op, operands)` + sentinel errors |
| `internal/api`              | HTTP + JSON           | `NewRouter`; everything else is unexported |
| `backend/api/openapi.yaml`  | the wire contract     | served at `/api/v1/openapi.yaml`, rendered at `/docs/` |
| `frontend/src/api`          | URL, JSON, Problem    | `calculate(request) → Promise<number>`     |
| `frontend/src/lib/engine`   | key semantics         | `press(state, key) → {state, compute?}`    |
| `useCalculator`             | ordering, I/O         | display text, history, `press(key)`        |
| components                  | presentation          | —                                          |

**REST with no redundancy.**
- The resource is a noun (`/calculations`), versioned under `/api/v1`, with
  explicit methods: wrong methods get `405` and an `Allow` header.
- The success body is only `{"result": n}`, because the client already knows
  what it sent. Errors carry only `code` and `detail`.
- Health is `204 No Content`.
- `POST` returns `200`, not `201`: nothing is stored, so there is no new
  resource to link to.
- `400` means malformed input; `422` means well-formed input that is
  mathematically invalid.
- Every endpoint has a consumer. The spec endpoint exists for the docs page;
  per-operation discovery endpoints were dropped because nothing used them.

**The backend owns the maths.** Every arithmetic result comes from the API,
including x² (`power` with exponent 2) and percent (`percentage`). The only
local operations are input editing: typing digits, backspace, and the
`+/−` sign toggle.

**A pure engine, a thin hook.** `lib/engine.ts` is a state machine with no
React and no I/O. When a key needs arithmetic, it returns a description of
the request and a function that continues from the result. This makes the
Windows behaviour exhaustively testable with plain function calls.

**No race conditions, by construction.**
- **One key at a time:** `useCalculator` processes keys strictly one at a
  time, in the order pressed. A key that needs the backend holds the queue
  until its response arrives, so at most one request is in flight. Every key
  sees the state left by the previous one, and responses cannot arrive out of
  order. Typing ahead during a slow request still works, and the keys apply
  in order afterwards.
- **C is the escape hatch:** it bypasses the queue, drops queued keys and
  invalidates the in-flight request with a generation counter, so a late
  response or error is ignored even if the network ignored the abort.
- **Timeouts:** requests time out after 1 s by racing a timer, which also
  releases the queue.
- **Backend:** it holds no mutable shared state, and the race detector runs
  in CI and in the Docker test stage.

**Windows fidelity where it helps users.**
- The layout and behaviour follow the Windows calculator, including 16-digit
  input and digit grouping. Error messages come from the backend instead, so
  they explain the problem.
- Keyboard shortcuts work too. Keys don't take focus when clicked, so Enter
  always means `=`, while Enter on a focused control (e.g. "Clear history")
  still activates it.
- Deliberate differences:
  - There are no memory keys.
  - xʸ replaces 1/x.
  - `%` is a two-number "percent of" operator (`15 % 200 = 30`), rather than
    the Windows rules where `%` rewrites the current entry.

**Operational hygiene.**
- Logging and resilience:
  - JSON structured logs with a per-request `X-Request-ID`.
  - Panic recovery that returns a Problem `500`.
  - Server timeouts, and graceful shutdown.
- Response headers: CSP (`default-src 'self'`; `data:` images are allowed only
  for Swagger UI's icons), `nosniff`, `Referrer-Policy`, and
  `Cache-Control: no-store` on the API.
- The container:
  - is distroless and runs as non-root
  - has a read-only root filesystem in Compose
  - runs a `HEALTHCHECK` through the binary's own `healthcheck` subcommand.
- One origin: the Go server serves the built frontend, so no CORS is needed.

### Assumptions

- Numbers are IEEE 754 doubles end to end. Values keep full precision between
  steps (`1 ÷ 3 × 3 = 1`); the display shows up to 15 significant digits.
- The service is stateless. History lives only in the browser session.
- Authentication and rate limiting belong to the gateway in front of the
  service, not to the service itself.

---

## How it was built

[`PROMPTS.md`](PROMPTS.md) summarises how the project evolved and lists the
prompts that shaped it, in order.
