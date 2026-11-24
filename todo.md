Oya\! 😂🔥 Now we're talking.

I want us to build something that will actually demonstrate your backend engineering ability, not another basic CRUD API that every beginner has on GitHub.

We'll build a **Wallet &amp; Transaction Management API in Go**, using only the Go standard library for the HTTP layer.

And I want you to stick with `net/http`\. No Gin, no Fiber, no GORM. This is an opportunity to understand how Go backend development works under the hood.

# Project: GopherWallet



(Portfolio project) (Go standard library) (Intermediate)

**A secure wallet API that allows users to manage balances, transfer funds, and view transaction histories with reliable transaction processing.**

## 1\. Technology stack

| Component | Technology |
|--------|--------|
| Language | Go |
| HTTP server | `net/http` |
| Routing | `http.ServeMux` |
| Database | PostgreSQL |
| PostgreSQL driver | pgx |
| Cache | Redis (later phase) |
| Authentication | JWT |
| Password hashing | bcrypt or Argon2id |
| Testing | `testing`, `httptest` |
| Containerization | Docker, Docker Compose |
| API documentation | OpenAPI/Swagger |

A note: pgx is a PostgreSQL driver, not an ORM. We'll write our SQL ourselves.

## 2\. Core functionality

We'll divide the application into modules.

### A. Authentication

- Register a user.
- Login.
- Logout.
- Get authenticated user's profile.
- Password hashing.
- JWT authentication middleware.

### B. Wallet management

Every user gets a wallet.

- Create wallet automatically during registration.
- Retrieve wallet balance.
- Retrieve wallet details.
- Prevent unauthorized access to another user's wallet.

### C. Transactions

This is the most important part of the project.

- Credit wallet.
- Debit wallet.
- Transfer funds between users.
- Retrieve transaction history.
- Retrieve individual transaction details.

### D. Transaction integrity

This is where we'll demonstrate real backend engineering.

Imagine User A has ₦10,000 and wants to transfer ₦7,000 to User B.

The system must guarantee\:

1. User A has sufficient funds.
2. User A is debited.
3. User B is credited.
4. Both operations succeed together or neither succeeds.
5. A transaction cannot be processed twice accidentally.

We'll use PostgreSQL transactions, row-level locking and idempotency keys.

**Important\:** Never use floating-point numbers for money. Store amounts as integer kobo (`BIGINT`\).

## 3\. Database design

We'll start with four tables.

### Users

| Column | Type | Purpose |
|--------|--------|--------|
| id | UUID | Primary key |
| email | VARCHAR | Unique email |
| password\_hash | TEXT | Hashed password |
| full\_name | VARCHAR | User's name |
| created\_at | TIMESTAMPTZ | Registration date |

### Wallets

| Column | Type | Purpose |
|--------|--------|--------|
| id | UUID | Primary key |
| user\_id | UUID | Wallet owner |
| currency | VARCHAR(3) | NGN |
| balance | BIGINT | Balance in kobo |
| created\_at | TIMESTAMPTZ | Creation date |

Each user should have one wallet.

### Transactions

| Column | Type | Purpose |
|--------|--------|--------|
| id | UUID | Transaction ID |
| reference | VARCHAR | Unique reference |
| wallet\_id | UUID | Associated wallet |
| type | VARCHAR | CREDIT / DEBIT |
| amount | BIGINT | Amount in kobo |
| status | VARCHAR | PENDING / SUCCESS / FAILED |
| description | TEXT | Transaction description |
| created\_at | TIMESTAMPTZ | Transaction date |

### Idempotency Keys

| Column | Type | Purpose |
|--------|--------|--------|
| id | UUID | Primary key |
| key | VARCHAR | Client-provided key |
| user\_id | UUID | Request owner |
| request\_hash | TEXT | Identifies request payload |
| response | JSONB | Stored response |
| created\_at | TIMESTAMPTZ | Creation date |

Add a unique constraint on the idempotency key within the appropriate user scope.

For transfers, we'll also create a `transfers` table to connect the debit and credit transaction records.

This gives us a proper audit trail instead of simply changing wallet balances.

## 4\. API endpoints

Here's the initial API contract.

### Authentication

| Method | Endpoint | Description |
|--------|--------|--------|
| POST | `/api/auth/register` | Register user |
| POST | `/api/auth/login` | Login |
| GET | `/api/users/me` | Current user |

### Wallet

| Method | Endpoint | Description |
|--------|--------|--------|
| GET | `/api/wallet` | Get wallet |
| GET | `/api/wallet/balance` | Get balance |

### Transactions

| Method | Endpoint | Description |
|--------|--------|--------|
| POST | `/api/transactions/credit` | Credit wallet |
| POST | `/api/transactions/debit` | Debit wallet |
| POST | `/api/transfers` | Transfer funds |
| GET | `/api/transactions` | List transactions |
| GET | `/api/transactions/{id}` | Get transaction |

We'll use Go's standard `ServeMux` method-aware routing, for example\:

```
mux.HandleFunc("POST /api/auth/register", authHandler.Register)
mux.HandleFunc("POST /api/transfers", transferHandler.Create)
mux.HandleFunc("GET /api/wallet", walletHandler.Get)
```



No external router required.

## 5\. Project architecture

I want you to avoid putting everything inside `main.go`\.

Use this structure\:

```
gopherwallet/
│
├── cmd/
│   └── api/
│       └── main.go
│
├── internal/
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   └── postgres.go
│   │
│   ├── user/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── model.go
│   │
│   ├── wallet/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── model.go
│   │
│   ├── transaction/
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── model.go
│   │
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── logging.go
│   │   └── recovery.go
│   │
│   └── server/
│       └── server.go
│
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_wallets.sql
│   ├── 003_create_transactions.sql
│   ├── 004_create_transfers.sql
│   └── 005_create_idempotency_keys.sql
│
├── tests/
│
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
└── README.md
```



### Understanding the architecture

We'll follow a simple layered architecture\:

**HTTP Handler**

Receives requests, validates input, sends responses

**Service Layer**

Business rules and transaction orchestration

**Repository Layer**

SQL queries and database operations

**PostgreSQL**

Persistent storage

For example, when a user transfers money\:

- Handler receives the transfer request.
- Service validates the amount and coordinates the operation.
- Repository executes the SQL transaction.
- PostgreSQL commits or rolls back the operation.

The handler shouldn't contain SQL queries. The repository shouldn't know anything about HTTP status codes.

## 6\. Development roadmap

I want you to work in milestones. Don't jump ahead because you saw Redis or JWT in the stack.

**Milestone 1 — Project foundation** (Start here)

Estimated: 1 day

- [ ] Initialize Go module.
- [ ] Create the project directory structure.
- [ ] Build an HTTP server with `net/http`\.
- [ ] Configure environment variables.
- [ ] Implement health-check endpoint.
- [ ] Add request logging middleware.
- [ ] Add panic recovery middleware.
- [ ] Implement graceful shutdown.

**Deliverable\:** A properly structured Go HTTP server.

---

**Milestone 2 — PostgreSQL integration**

Estimated: 1–2 days

- [ ] Set up PostgreSQL.
- [ ] Connect using pgx.
- [ ] Implement database migrations.
- [ ] Create user and wallet repositories.
- [ ] Implement registration.
- [ ] Automatically create a wallet for every user.

**Deliverable\:** Users and wallets persist in PostgreSQL.

---

**Milestone 3 — Authentication**

Estimated: 1 day

- [ ] Hash passwords.
- [ ] Implement login.
- [ ] Generate JWT tokens.
- [ ] Create authentication middleware.
- [ ] Protect wallet endpoints.

**Deliverable\:** Authenticated users can access their own wallets.

---

**Milestone 4 — Wallet operations**

Estimated: 1–2 days

- [ ] Get wallet balance.
- [ ] Credit wallet.
- [ ] Debit wallet.
- [ ] Validate transaction amounts.
- [ ] Prevent negative balances.
- [ ] Record every transaction.

**Deliverable\:** Working wallet operations.

---

**Milestone 5 — Transfers and consistency**

Estimated: 2 days

- [ ] Transfer between users.
- [ ] Implement PostgreSQL transactions.
- [ ] Implement row-level locking.
- [ ] Implement idempotency.
- [ ] Handle concurrent transfers.
- [ ] Add transaction history and pagination.

**Deliverable\:** Reliable transfer processing.

---

**Milestone 6 — Testing and deployment**

Estimated: 1–2 days

- [ ] Unit tests.
- [ ] HTTP handler tests with `httptest`\.
- [ ] PostgreSQL integration tests.
- [ ] Concurrent transfer tests.
- [ ] Dockerize application.
- [ ] Set up Docker Compose.
- [ ] Deploy to a VPS.

**Deliverable\:** A deployed and documented API.

---

**Milestone 7 — Advanced improvements**

After the core project works

- [ ] Redis caching.
- [ ] Rate limiting.
- [ ] Structured logging.
- [ ] Metrics and health monitoring.
- [ ] OpenAPI documentation.
- [ ] CI/CD with GitHub Actions.

These are estimates, not deadlines. The goal is to understand what you're implementing rather than rush through checkboxes.

## 7\. Important engineering challenges to solve

These are the things that will make this project stand out during a technical interview.

**A. Race conditions**

Imagine two requests arrive simultaneously\:

- Current balance: ₦10,000.
- Request 1 wants to withdraw ₦8,000.
- Request 2 also wants to withdraw ₦8,000.

Both requests might read ₦10,000 before either updates the database.

You must prevent both from succeeding.

Learn and implement PostgreSQL row-level locking with `SELECT ... FOR UPDATE`, or an appropriately atomic conditional update.

**B. Idempotency**

What happens when a client sends a transfer request, but the network times out before receiving the response?

The client retries.

Without idempotency, the user might get charged twice.

Your implementation should return the original result for a repeated key and reject reuse of the same key with a different request payload.

**C. Atomicity**

A transfer must never debit one wallet without crediting the other.

Use one database transaction\:

```
BEGIN

Lock sender and receiver wallets
Validate sender balance
Debit sender
Credit receiver
Record debit transaction
Record credit transaction
Record transfer

COMMIT
```



If anything fails\:

```
ROLLBACK
```



For concurrent transfers between the same wallets, acquire locks in a consistent wallet-ID order to reduce deadlocks.

**D. Auditability**

Never simply update a wallet balance and forget what happened.

Every movement of funds must have a corresponding transaction record. Ideally, introduce a double-entry ledger as an advanced improvement so that every transfer has balanced debit and credit entries.

One more thing: this is a portfolio simulation. Don't expose an unrestricted credit endpoint in a real production financial application.

## 8\. How to start today

Let's get your environment ready.

Since you're already using Ubuntu, open your terminal\:

**Step 1 — Create the project**

```
mkdir gopherwallet
cd gopherwallet

go mod init github.com/Covenantmondei/gopherwallet
```



**Step 2 — Create your directories**

```
mkdir -p cmd/api
mkdir -p internal/{config,database,user,wallet,transaction,middleware,server}
mkdir -p migrations
mkdir -p tests
```



**Step 3 — Create your first server**

Inside `cmd/api/main.go`\:

```
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

func main() {
    mux := http.NewServeMux()

    mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        json.NewEncoder(w).Encode(map[string]string{
            "status": "ok",
            "service": "gopherwallet",
        })
    })

    log.Println("Server running on :8080")

    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}
```



**Step 4 — Run it**

```
go run ./cmd/api
```



Visit\:

```
http://localhost:8080/health
```



Your first milestone is complete when the endpoint works and you've written a test for it using `httptest`\.

Note: This is just the initial bootstrap. We'll replace `ListenAndServe` with a properly configured `http.Server` that supports graceful shutdown in the foundation milestone.

## 9\. GitHub presentation

Your repository should eventually have\:

- A detailed README.
- Architecture diagram.
- Database ER diagram.
- API endpoint documentation.
- Setup instructions.
- Docker deployment instructions.
- Test instructions.
- Examples of successful and failed transactions.
- Explanation of concurrency and idempotency decisions.

These details will help someone reviewing your GitHub understand the engineering behind the project.
