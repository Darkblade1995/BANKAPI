# BANKAPI

██████╗  █████╗ ███╗   ██╗██╗  ██╗ █████╗ ██████╗ ██╗
██╔══██╗██╔══██╗████╗  ██║██║ ██╔╝██╔══██╗██╔══██╗██║
██████╔╝███████║██╔██╗ ██║█████╔╝ ███████║██████╔╝██║
██╔══██╗██╔══██║██║╚██╗██║██╔═██╗ ██╔══██║██╔═══╝ ██║
██████╔╝██║  ██║██║ ╚████║██║  ██╗██║  ██║██║     ██║
╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝

**Production-grade banking API built from scratch**  
`Go 1.23` · `PostgreSQL 16` · `Redis 7` · `React` · `WebSockets` · `Docker`

---

## What is BANKAPI?

BANKAPI is a production-grade banking REST API engineered entirely from scratch — no boilerplate generators, no magic frameworks. Every component was deliberately designed and implemented by hand, from ACID-compliant atomic transfers to real-time WebSocket notifications and a full React frontend.

This is not a CRUD tutorial. It is a production-grade architecture that mirrors what fintech companies use to handle real financial operations: double-token JWT authentication with rotation, Redis-backed brute-force protection, currency conversion with intelligent caching, structured JSON logging, and a complete test pyramid from unit to E2E.

---

## Architecture Overview

┌─────────────────────────────────────────────────────────────────────┐
│                         BANKAPI ARCHITECTURE                        │
│                                                                     │
│  React + TypeScript + Vite                                          │
│  Zustand · Axios · Tailwind CSS v4 · Recharts                      │
│  localhost:5173                                                     │
│       │                                                             │
│       │  HTTP + WebSocket (CORS)                                    │
│       ▼                                                             │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Go + Gin — :8080                         │   │
│  │                                                             │   │
│  │  CORS → Recovery → RateLimit → Logger → Auth               │   │
│  │                  (middleware chain)                         │   │
│  │                                                             │   │
│  │  /health          /swagger        /v1/...                  │   │
│  │  HealthCheck      Swagger UI      Business Logic           │   │
│  └──────────────────┬──────────────────────┬────────────────┘   │
│                     │                      │                       │
│         ┌───────────▼──────────┐  ┌────────▼──────────┐          │
│         │    PostgreSQL 16     │  │     Redis 7        │          │
│         │    Docker :5434      │  │    Docker :6380    │          │
│         │                     │  │                    │          │
│         │  users              │  │  rates:COP  TTL 1h │          │
│         │  accounts           │  │  failed_attempts   │          │
│         │  transactions       │  │  locked:email      │          │
│         │  refresh_tokens     │  │                    │          │
│         └─────────────────────┘  └────────────────────┘          │
└─────────────────────────────────────────────────────────────────────┘

---

## Why This Stack?

| Decision | Rationale |
|---|---|
| **Go + Gin** | Compiled, statically typed, native concurrency via goroutines. Gin handles 100k+ req/s on modest hardware |
| **PostgreSQL** | ACID compliance is non-negotiable for financial data. MVCC enables high concurrency without sacrificing consistency |
| **Redis** | In-memory operations for rate limiting and caching are 100-1000x faster than PostgreSQL for these use cases |
| **JWT double token** | Short-lived access tokens (15m) minimize exposure window. Refresh token rotation invalidates stolen tokens immediately |
| **Bcrypt cost=10** | Deliberately slow: ~100ms per hash. Makes brute-force attacks computationally prohibitive |
| **Goroutines for I/O** | Email and WebSocket notifications run in parallel goroutines — zero latency added to HTTP response |

---

## Features

### Security
- JWT double token: access token (15 min) + refresh token (7 days) with **rotation**
- Bcrypt password hashing with cost factor 10
- Account lockout after 3 failed login attempts (Redis TTL 30 min)
- Rate limiting per IP using Token Bucket algorithm (`rate.NewLimiter(10, 20)`)
- Real logout: refresh token deleted from DB, not just cleared client-side

### Banking
- Multi-currency accounts (COP, USD, EUR)
- Real-time currency conversion via ExchangeRate API
- Redis cache for exchange rates (TTL 1 hour) — zero external calls on cache hit
- **ACID atomic transfers**: `BEGIN → verify funds → debit → credit → INSERT transaction → COMMIT`
- Paginated transaction history with `total`, `page`, `limit`, `totalPages`

### Infrastructure
- Structured JSON logging with Uber Zap
- Swagger UI auto-generated from code annotations
- Gorilla WebSocket Hub pattern — real-time notifications on deposit/transfer
- Graceful shutdown: drains active connections before exit
- Health check endpoint verifying PostgreSQL + Redis connectivity
- Database migrations with golang-migrate (5 versioned migrations)
- Transactional emails via Resend (welcome, login alert, deposit, transfer, security alert)

### Testing
- **Unit tests** (24): JWT generation/validation, currency math, input validation
- **Integration tests** (11): SQL operations against real PostgreSQL via Testcontainers
- **E2E tests** (6): Full HTTP flows against real server + real DB + real Redis
- **GitHub Actions CI**: lint → unit → integration → E2E → build → security scan

---

## Atomic Transfer — The Core

Every transfer executes as a single PostgreSQL transaction:

```sql
BEGIN;
  SELECT balance FROM accounts WHERE id = $fromId;  -- verify funds
  UPDATE accounts SET balance = balance - $amount WHERE id = $fromId;
  UPDATE accounts SET balance = balance + $converted WHERE id = $toId;
  INSERT INTO transactions (...) VALUES (...);
COMMIT;
```

`balance = balance - $amount` is a single atomic PostgreSQL operation. Two concurrent transfers cannot read the same balance and both succeed — PostgreSQL serializes the operations internally. This eliminates the race condition that plagues naive implementations.

On any failure, `defer tx.Rollback()` fires automatically. Money never disappears.

---

## Refresh Token Rotation

WITHOUT ROTATION:
refreshToken A → new accessToken
refreshToken A → new accessToken  ← stolen token still works forever
WITH ROTATION:
refreshToken A → new accessToken + refreshToken B  ← A is deleted
refreshToken B → new accessToken + refreshToken C  ← B is deleted
refreshToken A → 401 Unauthorized  ← stolen token is dead

Every `/auth/refresh` call deletes the old refresh token and issues a new one. A stolen token becomes useless the moment the legitimate user makes any refresh call.

---

## Redis — Three Distinct Use Cases

┌─────────────────────────────────────────────────────────┐
│  KEY                    VALUE           TTL             │
├─────────────────────────────────────────────────────────┤
│  rates:COP              JSON ~170fx     1 hour          │
│  failed_attempts:email  integer         30 min (reset)  │
│  locked:email           true            30 min          │
└─────────────────────────────────────────────────────────┘

**Exchange rates**: `GET rates:COP` in microseconds vs. 200ms HTTP call to external API. Cache miss triggers a fresh API call and repopulates Redis.

**Brute-force protection**: `INCR failed_attempts:email` is atomic in Redis — two simultaneous wrong passwords cannot both read `2` and both write `3`. The counter is always correct under concurrency.

**Account lockout**: A separate key prevents authenticated requests even after Redis restart (TTL persists).

---

## WebSocket Hub Pattern

                HUB
                 │
    ┌────────────┼────────────┐
    │            │            │

Client{userId:1}  Client{2}  Client{3}
Conn + Send chan  ...         ...
Transfer happens →
hub.SendToUser(1, "transfer_sent", {...})   // sender
hub.SendToUser(2, "transfer_received", {...}) // receiver
Each Client has a dedicated WritePump goroutine.
All sends go through the Send channel — never directly to the WebSocket.
This serializes writes per-client, eliminating concurrent write panics.

---

## Token Bucket Rate Limiter

```go
rate.NewLimiter(10, 20)
//              │   └── burst capacity: 20 requests
//              └────── refill rate: 10 requests/second

// Per-IP bucket stored in sync.Map
// Attacker from single IP: bucket empties in 2 seconds → 429
// Legitimate users: bucket refills faster than consumed → never blocked
```

---

## Project Structure

BANKAPI/
├── cmd/
│   ├── api/
│   │   ├── main.go          # application bootstrap
│   │   ├── server.go        # graceful shutdown
│   │   ├── routes.go        # middleware chain + routing
│   │   ├── middleware.go    # rate limit, auth, logger
│   │   ├── jwt.go           # token generation/validation
│   │   ├── users.go         # register, login, refresh, logout
│   │   ├── accounts.go      # CRUD, deposit, withdraw, transfer
│   │   └── healthcheck.go   # /health endpoint
│   └── migrate/
│       └── main.go          # up/down migration runner
├── internal/
│   ├── database/
│   │   ├── db.go            # connection
│   │   ├── models.go        # Models aggregate
│   │   ├── users.go         # UserModel + bcrypt
│   │   ├── accounts.go      # AccountModel + Transfer ACID
│   │   └── tokens.go        # TokenModel + rotation
│   ├── cache/
│   │   └── cache.go         # Redis: rates, attempts, locks
│   ├── currency/
│   │   └── converter.go     # ExchangeRate API + cache-aside
│   ├── mailer/
│   │   └── mailer.go        # Resend transactional emails
│   ├── websocket/
│   │   ├── hub.go           # Hub + Client + WritePump
│   │   └── handler.go       # WebSocket upgrade handler
│   └── env/
│       └── env.go           # typed env var helpers
├── migrations/
│   ├── 000001_create_users.{up,down}.sql
│   ├── 000002_create_accounts.{up,down}.sql
│   ├── 000003_create_transactions.{up,down}.sql
│   ├── 000004_add_currency_to_transactions.{up,down}.sql
│   └── 000005_create_refresh_tokens.{up,down}.sql
├── frontend/                # React + TypeScript + Vite
├── cmd/api/tests/
│   ├── unit/                # 24 pure function tests
│   ├── integration/         # 11 DB tests via Testcontainers
│   └── e2e/                 # 6 full HTTP flow tests
├── .github/workflows/
│   └── ci.yml               # lint → test → build → security
├── Dockerfile               # multi-stage Alpine build
├── docker-compose.yml       # PostgreSQL + Redis
├── .env.example
└── README.md

---

## Database Schema

users
id · first_name · last_name · email (UNIQUE) · password · created_at
accounts
id · user_id (FK→users) · balance (BIGINT) · currency · created_at
transactions
id · from_account_id (FK) · to_account_id (FK) · amount
from_currency · to_currency · exchange_rate · converted_amount · created_at
refresh_tokens
id · user_id (FK→users) · token (UNIQUE) · expires_at · created_at


`balance` is stored as `BIGINT` (integer cents), never `FLOAT`. Floating-point arithmetic on monetary values introduces rounding errors that compound across millions of transactions.

---

## API Reference

### Public
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/v1/users/register` | Create account |
| `POST` | `/v1/users/login` | Authenticate → JWT pair |
| `POST` | `/v1/auth/refresh` | Rotate refresh token |
| `POST` | `/v1/auth/logout` | Invalidate refresh token |
| `GET` | `/health` | PostgreSQL + Redis status |
| `GET` | `/swagger/index.html` | Interactive API docs |

### Protected (Bearer JWT)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/v1/ws` | WebSocket connection |
| `POST` | `/v1/accounts` | Create bank account |
| `GET` | `/v1/accounts` | List user accounts |
| `GET` | `/v1/accounts/:id` | Get account detail |
| `POST` | `/v1/accounts/:id/deposit` | Deposit funds |
| `POST` | `/v1/accounts/:id/withdraw` | Withdraw funds |
| `POST` | `/v1/transfers` | Atomic transfer |
| `GET` | `/v1/accounts/:id/transactions` | Paginated history |

---

## Getting Started

### Prerequisites
- Go 1.23+
- Docker + Docker Compose
- Node.js 18+

### Run

```bash
# 1. Clone
git clone https://github.com/Darkblade1995/BANKAPI.git
cd BANKAPI

# 2. Environment
cp .env.example .env
# Edit .env with your API keys

# 3. Infrastructure
docker compose up -d

# 4. Migrations
go run cmd/migrate/main.go up

# 5. Backend
air  # hot reload
# or: go run ./cmd/api

# 6. Frontend (new terminal)
cd frontend && npm install && npm run dev
```

### Run Tests

```bash
# Unit (fast, no dependencies)
go test ./cmd/api/tests/unit/... -v

# Integration (requires Docker)
go test ./cmd/api/tests/integration/... -v -timeout 120s

# E2E (requires Docker)
go test ./cmd/api/tests/e2e/... -v -timeout 300s
```

---

## Environment Variables

```env
PORT=8080
DB_HOST=localhost
DB_PORT=5434
DB_USER=bankapi
DB_PASSWORD=bankapi123
DB_NAME=bankapi
JWT_SECRET=your-super-secret-jwt-key-change-this
EXCHANGE_API_KEY=your-exchangerate-api-key
RESEND_API_KEY=your-resend-api-key
RESEND_FROM=onboarding@resend.dev
REDIS_ADDR=localhost:6380
```

---

## Design Decisions

**Why store balance as BIGINT instead of DECIMAL?**  
`FLOAT` arithmetic introduces rounding errors that compound across millions of transactions. `DECIMAL` is accurate but slower. `BIGINT` storing integer cents is the industry standard: exact arithmetic, maximum performance, zero ambiguity.

**Why Redis for rate limiting instead of in-memory?**  
In-memory rate limiting doesn't survive restarts and doesn't work across multiple server instances. Redis persists across restarts and is shared by all instances — the correct solution for any production deployment.

**Why Testcontainers instead of mocks for integration tests?**  
Mocks test that your code calls the right methods, not that your SQL actually works. Testcontainers spin up a real PostgreSQL instance per test suite, run real queries, and verify real behavior — including index usage, constraint enforcement, and transaction isolation.

**Why Zap over the standard library logger?**  
The standard library logger produces unstructured text. Zap produces structured JSON compatible with every log aggregation platform (Datadog, Grafana Loki, ELK). The typed API (`zap.String`, `zap.Int`) avoids reflection, making it 10x faster than alternatives like Logrus under high load.

**Why graceful shutdown?**  
A hard kill (`SIGKILL`) drops in-flight requests mid-execution. A transfer interrupted between the debit and credit steps would leave the database in an inconsistent state. Graceful shutdown stops accepting new connections and waits for active requests to complete — a requirement for any system handling financial operations.

---

## CI/CD Pipeline


push / pull_request → main
│
├── Lint (golangci-lint)
├── Unit Tests
├── Integration Tests (Testcontainers)
├── E2E Tests (Testcontainers)
├── Build (go build)
└── Security Scan (govulncheck)

All jobs run in parallel. Build only triggers after all test jobs pass.

---

## Author

**Luis Fernando Agamez Atehortúa**  
Backend Engineer — Barranquilla, Colombia

- GitHub: [@Darkblade1995](https://github.com/Darkblade1995)
- YouTube: [@programandoconlucho](https://youtube.com/@programandoconlucho)
- LinkedIn: [luis-fernando-agamez](https://linkedin.com/in/luis-fernando-agamez)

---

## License

MIT License — see LICENSE for details.

---

*Every component exists for a reason. Nothing is arbitrary.*
