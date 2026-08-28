# Go Ledger
 
A ledger is how organizations do their bookeeping for financial transactions,  where money came
from, where it went, and a permanent record proving it happened.
 
This is a ledger engine built in Go that processes thousands of transactions per
second with full ACID guarantees. Serializable isolation prevents balance corruption
under concurrent load and every transaction is all or nothing. Everything either happens or fully rolls back. Money cannot be lost, duplicated, or corrupted. Even
if the server goes down mid-transaction.
 
---
 
## Design Decisions
 
### Why Serializable Isolation Instead of SELECT FOR UPDATE
 
The obvious approach for preventing concurrent balance corruption is locking the
account row with SELECT FOR UPDATE before reading the balance. But this only works
when the balance is stored on the account row. In this ledger the balance is never
stored, it is calculated from the entry history on demand. Locking the account row
does not prevent another transaction from inserting entries that change the balance
between your read and your write.
 
Serializable isolation solves this correctly. PostgreSQL tracks what each transaction
read and detects conflicts at commit time. If two concurrent transactions would have
produced an incorrect result one is aborted and retried automatically.
 
The tradeoff is retries under high concurrency. Under load testing with ten
simultaneous goroutines hitting the same account, three retries were insufficient with
five of ten transactions failed. Increasing to ten retries resolved this. The retry
count is a tunable parameter based on expected concurrency.
 
### Why Balance Is Derived Not Stored
 
The simple approach is storing a balance column on the account and updating it with
every transaction. This creates two sources of truth, the balance field and the
entry history. Under any failure scenario they can diverge. A server crash between
writing entries and updating the balance leaves the system inconsistent with no way
to know which value is correct.
 
By deriving balance from entries on demand there is only one source of truth. The
balance is always the sum of all entries for an account. It cannot be wrong because
it does not exist independently of the data that defines it.
 
As a side effect you get point in time balance reconstruction for free. Query the
sum of entries up to any timestamp and you know exactly what the balance was at that
moment in history.
 
The tradeoff is query performance at extreme entry counts. A balance query scans all
entries for an account since the beginning of time. A composite index on
(account_id, created_at) keeps this fast at current scale.
 
 
## Architecture
 
**Database Layer** — PostgreSQL running in Docker for portability. Schema designed
around three tables — accounts, transactions, and entries. All queries are written
in raw SQL and compiled into type safe Go code using sqlc. 
 
**Service Layer** — where the business logic lives. Before any transaction reaches
the database it passes through a validation pipeline, minimum entries check, balance
check, idempotency lookup, sufficient funds verification. The actual database write
happens inside a serializable PostgreSQL transaction with automatic retry on
serialization failure. 
 
**API Layer** — a lightweight REST API built with Go's standard net/http library.
Four endpoints expose the core ledger functionality. 
 
I chose Go because its concurrency model maps naturally to the problem. Goroutines
handle concurrent transaction submissions. The language's performance characteristics
and straightforward concurrency primitives made it the right tool for building
something that has to be both fast and correct.
 
---
 
## Performance
 
Benchmarked on Apple M1 Pro with PostgreSQL 16 running in Docker Desktop.
 
**Baseline**
 
```
Sequential:  1,450,331 ns/op    241 allocs/op    ~689 TPS
Parallel:      461,243 ns/op    239 allocs/op    ~17,000 TPS
```
 
**Profiling**
 
CPU profiling with pprof showed 93% of time spent waiting for PostgreSQL — syscalls
and kernel event notification. Application code consumed less than 5% of CPU time.
The bottleneck was not computation but database round trips.
 
Memory profiling identified two hotspots. GetAccount calls contributed 28% of
cumulative memory, one database round trip per entry in the validation loop.
CreateEntry calls contributed 45% — one INSERT per entry with full query encoding
and result scanning overhead.
 
**Optimizations**
 
First, replaced per-entry GetAccount calls with a single batch query using
PostgreSQL's ANY operator. One round trip to fetch all accounts regardless of entry
count.
 
Second, replaced individual INSERT statements with PostgreSQL's binary COPY protocol
via pgx CopyFrom. All entries inserted in one round trip. Eliminated per-row query
parsing and result scanning.
 
Third, empirically determined optimal connection pool size of 25 by benchmarking at
10, 25, 50, and 100 connections. Performance degraded beyond 25 — PostgreSQL backend
process overhead exceeded the benefit of additional parallelism.
 
**Results**
 
```
Sequential:  1,083,481 ns/op    228 allocs/op    ~923 TPS     (+34%)
Parallel:      356,330 ns/op    226 allocs/op    ~22,400 TPS  (+32%)
```
 
 
---
 
## Correctness Guarantees
 
Every guarantee below is verified by a test — not just claimed.
 
**Double Entry Invariant**
Every transaction must have equal total debits and credits. Money cannot appear or
disappear. Enforced at the service layer before any database call and at the database
level through CHECK constraints.
 
Verified by `TestCreateTransaction_Balance` — unbalanced transactions are rejected
before touching the database.
 
**Atomic Writes**
Every transaction either fully commits or fully rolls back. A crash between writing
the transaction row and writing the entries leaves no orphaned data.
 
Verified by `TestCreateTransaction_ChaosTest` — deliberately fails mid-write using
a non-existent account ID and confirms zero transaction rows exist in the database
afterward.
 
**Idempotency**
Submitting the same transaction multiple times produces the same result as submitting
it once. No duplicate writes occur regardless of how many times the same request is
submitted.
 
Verified by `TestCreateTransaction_IdempotencyRetry` — submits the same transaction
twice and asserts identical transaction IDs are returned with only one row in the
database.
 
**Concurrent Correctness**
Account balances are never corrupted under concurrent load. Ten goroutines
withdrawing from the same account simultaneously produce a mathematically correct
final balance.
 
Verified by `TestCreateTransaction_ConcurrentWithdraw` — ten concurrent withdrawals
with final balance assertion. Passes consistently with `go test -race` detecting zero
data races.
 
**Sufficient Funds**
Asset accounts cannot be overdrawn. A transaction that would take an account below
zero is rejected before any write occurs.
 
Verified by `TestCreateTransaction_InsufficientFunds` — attempts to credit more than
the current balance and asserts `ErrInsufficientFunds` is returned.
 
**Point In Time Reconstruction**
The balance of any account at any historical moment can be reconstructed by summing
entries up to that timestamp. This is a structural guarantee — balance is always
derived from entries, never stored independently. The two cannot disagree because
only one exists.
 
---
 
## Running Locally

## Makefile

Common commands are available via Make.

```bash
make run          # start the HTTP server
make cli          # run the CLI
make test         # run the full test suite with race detector
make bench        # run benchmarks
make build        # compile both binaries to bin/
make migrate      # run database migrations
make docker-up    # start PostgreSQL in Docker
make docker-down  # stop PostgreSQL
```

**Prerequisites**
 
Go 1.22 or later and Docker Desktop.
 
**Setup**
 
```bash
git clone https://github.com/rgarcia2304/go_ledger
cd go_ledger
 
docker compose up -d
 
goose -dir migrations postgres "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable" up
 
go run ./cmd/server/...
```
 
Open `http://localhost:8080` for the live demo.
Open `http://localhost:8080/api` for the manual API explorer.
 
**Run the test suite**
 
```bash
go test -race ./...
```
 
**Run benchmarks**
 
```bash
go test -bench=. -benchtime=10s -benchmem ./internal/ledger/...
```
 
**CLI usage**
 
```bash
go run ./cmd/cli/... create-account --name "Cash" --type asset --currency USD
 
go run ./cmd/cli/... create-transaction --file transaction.json
 
go run ./cmd/cli/... get-balance --accountID "your-account-uuid"
 
go run ./cmd/cli/... get-transaction-history --accountID "your-account-uuid"
```
 
**Transaction file format**
 
```json
{
    "description": "Payment for services",
    "occurred_at": "2026-04-16T10:00:00Z",
    "entries": [
        {
            "account_id": "asset-account-uuid",
            "amount_cents": 10000,
            "direction": "debit",
            "currency": "USD"
        },
        {
            "account_id": "revenue-account-uuid",
            "amount_cents": 10000,
            "direction": "credit",
            "currency": "USD"
        }
    ]
}
```
 
The idempotency key is derived automatically from the SHA256 hash of the file
contents. Submitting the same file twice returns the existing transaction without
creating a duplicate.
 
---
 
## What's Next
 
Building this taught me a lot about database internals, concurrency design, and what
it actually takes to make a financial system correct rather than just functional. The
gap between those two things turned out to be the interesting part.
 
If I were to extend this project the natural next step is an event driven
architecture. Modern banks like Monzo and Nubank decouple transaction ingestion from
processing using message queues. In their systems the transaction is accepted immediately and
processed asynchronously. This enables higher throughput and better fault tolerance
than the synchronous request response model this ledger uses today. The outbox
pattern would be the starting point for making that transition safely.
 
This ledger could also be the foundation of a larger banking application. By adding
users, authentication, multi-currency support, and a full API gateway with rate
limiting and request signing.
 
The honest limitations of this project are scale and failure handling. The chaos
testing proves atomicity under a simulated crash but a production system needs more
 automated failover to a standby database, point in time recovery from backups,
geographic redundancy across availability zones, and observability infrastructure
that pages someone when something goes wrong at 3am. Those are real operational
concerns that a solo portfolio project can simulate but not fully solve.
 
The benchmark numbers were measured on a development machine with a containerized
database. Production hardware with a dedicated PostgreSQL instance, tuned server
configuration, and a read replica for balance queries would produce significantly
higher throughput.
 
---
 
## Tech Stack
 
Go 1.22, PostgreSQL 16, sqlc, pgx, goose, Docker, Cobra
