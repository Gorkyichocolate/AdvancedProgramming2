# 🚀 Quick Reference - Микросервисы

## 📍 Порты

| Сервис | REST | gRPC | Назначение |
|--------|------|------|-----------|
| **Order Service** | 8001 | 9001 | Управление заказами |
| **Payment Service** | 8089 | 9002 | Обработка платежей |
| **Notification Service** | 8003 | - | Уведомления |
| **Redis** | 6379 | - | Кэширование |
| **RabbitMQ** | 15672 (UI) | 5672 | Message Queue |

---

## 🔧 Быстрый старт

### 🐳 Вариант 1: Docker Compose (с Redis, RabbitMQ)
```bash
docker-compose up --build
# Запускает все сервисы:
# - Order Service (8001/9001)
# - Payment Service (8089/9002)
# - Notification Service (8003)
# - PostgreSQL (5432, БД: ap2 + postgres)
# - Redis (6379)
# - RabbitMQ (5672, UI: 15672)

# Проверка статуса
docker-compose ps
```

### 🚀 Вариант 2: Локально (БД на localhost)

#### 0️⃣ Подготовить БД (один раз)
```bash
# Создать БД для заказов
createdb -U biba ap2
psql -U biba -d ap2 << 'EOF'
CREATE TABLE IF NOT EXISTS orders
(
    id          UUID      PRIMARY KEY,
    customer_id TEXT      NOT NULL,
    item_name   TEXT      NOT NULL,
    amount      BIGINT    NOT NULL CHECK (amount > 0),
    status      TEXT      NOT NULL CHECK (status IN ('Pending', 'Paid', 'Failed', 'Cancelled')),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
EOF

# Создать БД для платежей
psql -U biba << 'EOF'
CREATE TABLE IF NOT EXISTS payments
(
    id             UUID        PRIMARY KEY,
    order_id       UUID        NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    status         TEXT        NOT NULL CHECK (status IN ('Authorized', 'Declined')),
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);
EOF
```

#### 1️⃣ Запустить Order Service
```bash
export PAYMENT_GRPC_ADDR=localhost:9002
export ORDER_ADDR=:8001
export ORDER_GRPC_ADDR=:9001
export ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
go run order-service/cmd/order-service/main.go
```

#### 2️⃣ Запустить Payment Service
```bash
export PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
export PAYMENT_ADDR=:8089
export PAYMENT_GRPC_ADDR=:9002
go run payment-service/cmd/payment-service/main.go
```

---

## 📡 REST API Quick Commands

### ✅ Complete Order & Payment Flow

#### 1️⃣ Создать заказ
```bash
ORDER=$(curl -s -L -X POST http://localhost:8001/orders/ \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":"cust-1",
    "item_name":"Laptop",
    "amount":50000
  }')

echo $ORDER | jq '.'
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-1",
  "item_name": "Laptop",
  "amount": 50000,
  "status": "Pending",
  "created_at": "2026-05-14T12:00:00Z"
}
```

#### 2️⃣ Проверить статус заказа (должен быть Pending)
```bash
curl -s http://localhost:8001/orders/$ORDER_ID | jq '.'
```

#### 3️⃣ Обработать платёж через REST
```bash
PAYMENT=$(curl -s -X POST http://localhost:8089/payments \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\":\"$ORDER_ID\",
    \"amount\":50000
  }")

PAYMENT_ID=$(echo $PAYMENT | jq -r '.id')
echo "💳 Payment processed: $PAYMENT_ID"
echo $PAYMENT | jq '.'
```

**Response (amount ≤ 50000 = Authorized):**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "txn-001-abc123",
  "amount": 50000,
  "status": "Authorized",
  "created_at": "2026-05-14T12:00:05Z"
}
```

#### 4️⃣ Проверить обновлённый статус заказа (должен быть Paid)
```bash
curl -s http://localhost:8001/orders/$ORDER_ID | jq '.'
# Status теперь: "Paid"
```

#### 5️⃣ Получить платёж
```bash
curl -s http://localhost:8089/payments/$ORDER_ID | jq '.'
```

---

### 🔴 Failed Payment (amount > 50000)

```bash
ORDER2=$(curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id":"cust-2",
    "item_name":"Server",
    "amount":100000
  }')

ORDER_ID2=$(echo $ORDER2 | jq -r '.id')

# Платёж будет отклонён (Declined)
PAYMENT2=$(curl -s -X POST http://localhost:8089/payments \
  -H "Content-Type: application/json" \
  -d "{
    \"order_id\":\"$ORDER_ID2\",
    \"amount\":100000
  }")

echo $PAYMENT2 | jq '.status'  # "Declined"

# Проверить статус заказа (остался Pending)
curl -s http://localhost:8001/orders/$ORDER_ID2 | jq '.status'  # "Pending"
```

---

### Order Service

```bash
# 🔍 Получить заказ
curl http://localhost:8001/orders/{order_id}

# ❌ Отменить заказ
curl -X PATCH http://localhost:8001/orders/{order_id}/cancel

# 📋 Список заказов с фильтрацией
curl "http://localhost:8001/orders?min_amount=1000&max_amount=50000"
```

### Payment Service

```bash
# 💳 Обработать платёж
curl -X POST http://localhost:8089/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":"{order_id}","amount":50000}'

# 🔍 Получить платёж
curl http://localhost:8089/payments/{order_id}
```

---

## 🔌 gRPC Quick Commands

### ProcessPayment
```bash
grpcurl -plaintext \
  -d '{"order_id":"{order_id}","amount":50000}' \
  localhost:9002 ap2.v1.PaymentService/ProcessPayment
```

### ListPayments
```bash
# 📊 Список авторизованных платежей
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:9002 ap2.v1.PaymentService/ListPayments

# 📊 Список отклонённых платежей
grpcurl -plaintext \
  -d '{"status":"Declined"}' \
  localhost:9002 ap2.v1.PaymentService/ListPayments
```

### SubscribeToOrderUpdates (Streaming)
```bash
grpcurl -plaintext \
  -d '{"order_id":"{order_id}"}' \
  localhost:9001 ap2.v1.OrderService/SubscribeToOrderUpdates
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

## 💾 Database Setup (Локальный запуск)

⚠️ **Важно**: Перед локальным запуском убедитесь, что БД создана и заполнена!

```bash
# 1. Создать БД для заказов
createdb -U biba ap2

# 2. Создать таблицу orders
psql -U biba -d ap2 << 'EOF'
CREATE TABLE IF NOT EXISTS orders
(
    id          UUID      PRIMARY KEY,
    customer_id TEXT      NOT NULL,
    item_name   TEXT      NOT NULL,
    amount      BIGINT    NOT NULL CHECK (amount > 0),
    status      TEXT      NOT NULL CHECK (status IN ('Pending', 'Paid', 'Failed', 'Cancelled')),
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
EOF

# 3. Создать таблицу payments (в default БД 'postgres')
psql -U biba << 'EOF'
CREATE TABLE IF NOT EXISTS payments
(
    id             UUID        PRIMARY KEY,
    order_id       UUID        NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    status         TEXT        NOT NULL CHECK (status IN ('Authorized', 'Declined')),
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);
EOF

# 4. Проверка
psql -U biba -d ap2 -c "\dt"  # Показать таблицы в БД ap2
psql -U biba -c "\dt"          # Показать таблицы в БД postgres
```

---

## 🔗 .env Template (локальный запуск)

```
# Order Service
ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
ORDER_ADDR=:8001
ORDER_GRPC_ADDR=:9001

# Payment Service
PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
PAYMENT_ADDR=:8089
PAYMENT_GRPC_ADDR=:9002

# Payment Client (Order Service → Payment Service)
PAYMENT_GRPC_ADDR=localhost:9002

# Redis (если используется)
REDIS_URL=localhost:6379
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

# ❌ Заказ создаётся но не появляется в БД
# → Таблицы не созданы! Выполните Database Setup инструкции выше

# ❌ ERROR: database "ap2" does not exist
# → Создайте БД: createdb -U biba ap2

# ❌ redis: connection pool: failed to dial
# → Redis не запущен. Запустите: redis-server
# → Или используйте docker-compose: docker-compose up
```

---

## 🧪 Complete Test Flow

```bash
# 1. Create order
ORDER_ID=$(curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"test","item_name":"Item","amount":5000}' | jq -r '.id')

echo "Order: $ORDER_ID"

# 2. Get order
curl http://localhost:8001/orders/$ORDER_ID | jq '.status'

# 3. List payments (gRPC)
grpcurl -plaintext -d '{"status":"Authorized"}' \
  localhost:9002 ap2.v1.PaymentService/ListPayments | jq '.payments | length'

# 4. Subscribe to updates (in background)
grpcurl -plaintext -d "{\"order_id\":\"$ORDER_ID\"}" \
  localhost:9001 ap2.v1.OrderService/SubscribeToOrderUpdates &

# 5. Cancel order
curl -X PATCH http://localhost:8001/orders/$ORDER_ID/cancel | jq '.status'
```

---

## � Redis & Cache Testing

### Проверка Redis
```bash
# Проверить если Redis запущен
redis-cli ping
# Output: PONG

# Посмотреть все ключи
redis-cli KEYS "*"

# Посмотреть значение кэша
redis-cli GET "order:{order_id}"

# Очистить весь кэш
redis-cli FLUSHALL
```

### ⚡ Test Cache-Aside Pattern
```bash
# 1. Создать заказ
ORDER_ID=$(curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cache-test","item_name":"Item","amount":5000}' | jq -r '.id')

# 2. Первый запрос (cache miss - идёт в БД)
time curl http://localhost:8001/orders/$ORDER_ID > /dev/null

# 3. Второй запрос (cache hit - должен быть быстрее)
time curl http://localhost:8001/orders/$ORDER_ID > /dev/null

# 4. Проверить значение в Redis (если используется)
redis-cli GET "order:$ORDER_ID" | jq '.'

# 5. Инвалидировать кэш (отмена заказа)
curl -X PATCH http://localhost:8001/orders/$ORDER_ID/cancel

# 6. Снова cache miss
time curl http://localhost:8001/orders/$ORDER_ID > /dev/null
```

### 🚀 Test Parallelism (Параллельные запросы)

#### Создание 100 заказов параллельно
```bash
# GNU Parallel (если установлен)
seq 1 100 | parallel -j 10 'curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d "{\"customer_id\":\"test-{}\",\"item_name\":\"Item-{}\",\"amount\":$((5000 + {}))}" | jq -r .id' > order_ids.txt

# Или через xargs
seq 1 100 | xargs -P 10 -I {} curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d "{\"customer_id\":\"test-{}\",\"item_name\":\"Item-{}\",\"amount\":$((5000 + {}))}" 
```

#### ⏱️ Параллельное чтение (10 одновременных запросов)
```bash
# Прочитать один заказ 10 раз параллельно и измерить время
time for i in {1..10}; do 
  curl -s http://localhost:8001/orders/$ORDER_ID > /dev/null &
done
wait

# Ожидаемый результат: параллельные запросы = быстрее, чем последовательные
# Cache hit должен быть ~5-10ms per request
```

#### 📊 Load Test с Apache Bench
```bash
# 1000 запросов, 20 параллельных
ab -n 1000 -c 20 http://localhost:8001/orders/$ORDER_ID

# Результат покажет:
# - Requests per second
# - Time per request (mean)
# - Failed requests (если > 0, значит проблема с параллелизмом)
```

#### 🔄 Stress Test с GNU Parallel
```bash
# Создать 50 заказов, потом читать каждый 20 раз параллельно
cat order_ids.txt | parallel -j 20 'for i in {1..20}; do 
  curl -s http://localhost:8001/orders/{} > /dev/null
done'

# Проверить Redis статистику
redis-cli INFO stats
# clients, commands_processed_per_sec
```

#### ✅ Проверка Redis под нагрузкой
```bash
# Мониторить Redis во время нагрузки (в отдельном терминале)
redis-cli MONITOR

# Или смотреть статистику
watch 'redis-cli INFO | grep -E "used_memory|connected_clients|keys_hit|keys_miss"'

# Запустить нагрузку в другом терминале
ab -n 5000 -c 50 http://localhost:8001/orders/$ORDER_ID
```

---

## �📖 Full Documentation

👉 [USAGE_GUIDE.md](USAGE_GUIDE.md) — Complete documentation with examples and explanations
