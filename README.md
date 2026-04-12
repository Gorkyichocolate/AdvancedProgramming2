# AdvancedProgramming2

Two-service Go microservice system with clear domain boundaries:
- `order-service` owns order lifecycle.
- `payment-service` owns payment authorization outcome.

External client traffic to `order-service` still uses REST, while internal communication from `order-service` to `payment-service` now uses gRPC.

This project uses a contract-first workflow with a separate protobuf repository (`github.com/Gorkyichocolate/Proto`) and a generated-code module (`github.com/Gorkyichocolate/ap2-generated`).

## Why This Architecture

### 1) Layered / Hexagonal-ish Service Structure
Each service is split into:
- `transport` (HTTP handlers, request/response mapping)
- `usecase` (business rules)
- `repository` (PostgreSQL persistence)
- `domain` (entities + domain constants)
- `app` (wiring dependencies + routes)

Decision rationale:
- keeps business rules independent from Gin and pgx
- enables focused testing by layer
- keeps infra decisions (DB/HTTP client) at application edge

### 2) Synchronous Inter-Service Call (Order -> Payment)
`order-service` creates an order first, then calls `payment-service` synchronously using gRPC.

Decision rationale:
- strict contract enforcement through Protocol Buffers
- immediate feedback to the caller about payment status
- lower complexity compared to async broker-based flows

Trade-off:
- coupling to payment availability/latency

### 3) Database per Service Boundary
`orders` and `payments` are separate schemas/tables, each owned by its service.

Decision rationale:
- enforces bounded context ownership
- avoids shared mutable domain model between services
- supports independent schema evolution

## Bounded Contexts

### Order Context (`order-service`)
Responsibilities:
- create order
- fetch order
- cancel pending order
- list orders by amount range

Aggregate/data:
- `Order` with status: `Pending | Paid | Failed | Cancelled`
- table: `orders`

Invariants:
- `amount > 0`
- only `Pending` orders can be cancelled

Public API:
- `POST /orders/`
- `GET /orders/:id`
- `PATCH /orders/:id/cancel`
- `GET /orders/getList?min_amount=...&max_amount=...`

### Payment Context (`payment-service`)
Responsibilities:
- process payment decision for order
- fetch payment by order id

Aggregate/data:
- `Payment` with status: `Authorized | Declined`
- table: `payments`

Business rule:
- if `amount > PaymentLimit (100000)` => `Declined`
- otherwise => `Authorized`

Public API:
- `POST /payments/`
- `GET /payments/:id` (where `:id` is `order_id`)

## Failure Handling

### 1) Payment service unavailable or timeout
Where: `order-service/internal/transport/payment_client.go`

Behavior:
- order is persisted first as `Pending`
- on payment call error/non-201, order is updated to `Failed`
- create-order response becomes `503 Service Unavailable` with order payload

Why:
- caller sees order id and final failure state
- system avoids leaving successful API response with unknown payment outcome

### 2) Validation failures
Examples:
- invalid create payload (`amount < 1`, missing fields) => `400`
- list filters missing/non-integer => `400`
- list range outside `[1000, 50000]` or `min > max` => `400`

### 3) Not found semantics
Examples:
- unknown order id => `404`
- unknown payment by order id => `404`
- empty list by filter in order list => `404`

### 4) Database constraint failures
Examples:
- malformed `order_id` for payments (`UUID` column) can produce insert error
- any repository error is surfaced as `500` (or as availability error path for payment call from order-service)

### 5) Idempotency on order creation
`order-service` caches responses by `Idempotency-Key` in memory.

Behavior:
- repeated request with same key returns cached response
- protects from duplicate create operations during retries

Trade-off:
- in-memory only (not shared across instances, lost on restart)

## Configuration

Environment variables:
- `ORDER_DB_URL` (required)
- `PAYMENT_DB_URL` (required)
- `ORDER_ADDR` (default `:8086`)
- `ORDER_GRPC_ADDR` (default `:8085`)
- `PAYMENT_ADDR` (default `:8087`)
- `PAYMENT_GRPC_ADDR` (default `:8088`)

Important operational note:
- `order-service` exposes REST on `ORDER_ADDR` and gRPC on `ORDER_GRPC_ADDR`
- `payment-service` exposes REST on `PAYMENT_ADDR` and gRPC on `PAYMENT_GRPC_ADDR`
- internal order-to-payment communication now uses `PAYMENT_GRPC_ADDR`

Recommended local setup:
- `PAYMENT_ADDR=:8087`
- `PAYMENT_GRPC_ADDR=:8088`
- `ORDER_ADDR=:8086`
- `ORDER_GRPC_ADDR=:8085`

## Database Migrations

Run these SQL files on your PostgreSQL database(s):
- `order-service/migrations/001_create_orders.sql`
- `payment-service/migrations/001_create_payments.sql`

## Run Locally

From repository root:

```bash
# Terminal 1: payment-service
export PAYMENT_DB_URL='postgres://user:pass@localhost:5432/payments_db?sslmode=disable'
export PAYMENT_ADDR=':8087'
export PAYMENT_GRPC_ADDR=':8088'
go run ./payment-service/cmd/payment-service

# Terminal 2: order-service
export ORDER_DB_URL='postgres://user:pass@localhost:5432/orders_db?sslmode=disable'
export ORDER_ADDR=':8086'
export ORDER_GRPC_ADDR=':8085'
export PAYMENT_GRPC_ADDR='localhost:8088'
go run ./order-service/cmd/order-service
```

## Example Requests

Create order:

```bash
curl -X POST http://localhost:8086/orders/ \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: abc-123' \
  -d '{"customer_id":"c1","item_name":"book","amount":1500}'
```

Get list by amount range:

```bash
curl 'http://localhost:8086/orders/getList?min_amount=1000&max_amount=50000'
```

## Future Improvements

- Introduce outbox + message broker for resilient async payment workflow
- Add distributed idempotency storage (Redis/Postgres)
- Standardize error contract and observability (structured logs, traces, metrics)
- Add integration tests for cross-service failure modes
