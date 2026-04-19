# 🚀 Quick Reference - Микросервисы

## 📍 Порты

| Сервис | REST | gRPC | Назначение |
|--------|------|------|-----------|
| **Order Service** | 8086 | 8085 | Управление заказами |
| **Payment Service** | 8087 | 8088 | Обработка платежей |

---

## 🔧 Быстрый старт

### 1️⃣ Запустить Order Service
```bash
export PAYMENT_GRPC_ADDR=localhost:8088
export ORDER_ADDR=:8086
export ORDER_GRPC_ADDR=:8085
export ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
go run order-service/cmd/order-service/main.go
```

### 2️⃣ Запустить Payment Service
```bash
export PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
export PAYMENT_ADDR=:8087
export PAYMENT_GRPC_ADDR=:8088
go run payment-service/cmd/payment-service/main.go
```

---

## 📡 REST API Quick Commands

### Order Service

```bash
# ➕ Создать заказ
curl -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-1","item_name":"Laptop","amount":50000}'

# 🔍 Получить заказ
curl http://localhost:8086/orders/{order_id}

# ❌ Отменить заказ
curl -X PATCH http://localhost:8086/orders/{order_id}/cancel

# 📋 Список заказов
curl "http://localhost:8086/orders?min_amount=1000&max_amount=50000"
```

### Payment Service

```bash
# 💳 Обработать платёж
curl -X POST http://localhost:8087/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":"{order_id}","amount":50000}'

# 🔍 Получить платёж
curl http://localhost:8087/payments/{order_id}
```

---

## 🔌 gRPC Quick Commands

### ProcessPayment
```bash
grpcurl -plaintext \
  -d '{"order_id":"{order_id}","amount":50000}' \
  localhost:8088 ap2.v1.PaymentService/ProcessPayment
```

### ListPayments
```bash
# 📊 Список авторизованных платежей
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments

# 📊 Список отклонённых платежей
grpcurl -plaintext \
  -d '{"status":"Declined"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments
```

### SubscribeToOrderUpdates (Streaming)
```bash
grpcurl -plaintext \
  -d '{"order_id":"{order_id}"}' \
  localhost:8085 ap2.v1.OrderService/SubscribeToOrderUpdates
```

---

## 📊 Статусы

### Order Statuses
| Статус | Значение |
|--------|----------|
| `Pending` | Ожидание платежа |
| `Paid` | Успешно оплачено |
| `Failed` | Платёж отклонён |
| `Cancelled` | Отменено пользователем |

### Payment Statuses
| Статус | Значение |
|--------|----------|
| `Authorized` | Авторизовано (amount ≤ 50000) |
| `Declined` | Отклонено (amount > 50000) |

---

## 💾 Database Setup

```bash
# Orders
psql -U biba -d ap2 << 'EOF'
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id TEXT NOT NULL,
    item_name TEXT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status TEXT NOT NULL CHECK (status IN ('Pending', 'Paid', 'Failed', 'Cancelled')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
EOF

# Payments
psql -U biba -d postgres << 'EOF'
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    status TEXT NOT NULL CHECK (status IN ('Authorized', 'Declined')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
EOF
```

---

## 🔗 .env Template

```
ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
ORDER_ADDR=:8086
ORDER_GRPC_ADDR=:8085
PAYMENT_ADDR=:8087
PAYMENT_GRPC_ADDR=:8088
PAYMENT_GRPC_ADDR=localhost:8088
```

---

## 📝 Full JSON Responses

### Order Response
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-001",
  "item_name": "Laptop",
  "amount": 50000,
  "status": "Paid",
  "created_at": "2026-04-19T23:30:00Z"
}
```

### Payment Response
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "txn-001-abc123",
  "amount": 50000,
  "status": "Authorized"
}
```

---

## 🐛 Common Errors

```bash
# ❌ 503 Service Unavailable
# → Payment Service не запущен или недоступен на PAYMENT_GRPC_ADDR

# ❌ 404 Not Found
# → Заказ/платёж не существует, проверьте ID

# ❌ 400 Bad Request
# → Невалидные параметры (amount <= 0, отсутствуют поля, order_id не UUID)
```

---

## 🧪 Complete Test Flow

```bash
# 1. Create order
ORDER_ID=$(curl -s -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"test","item_name":"Item","amount":5000}' | jq -r '.id')

echo "Order: $ORDER_ID"

# 2. Get order
curl http://localhost:8086/orders/$ORDER_ID | jq '.status'

# 3. List payments (gRPC)
grpcurl -plaintext -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments | jq '.payments | length'

# 4. Subscribe to updates (in background)
grpcurl -plaintext -d "{\"order_id\":\"$ORDER_ID\"}" \
  localhost:8085 ap2.v1.OrderService/SubscribeToOrderUpdates &

# 5. Cancel order
curl -X PATCH http://localhost:8086/orders/$ORDER_ID/cancel | jq '.status'
```

---

## 📖 Full Documentation

👉 [USAGE_GUIDE.md](USAGE_GUIDE.md) — Complete documentation with examples and explanations
