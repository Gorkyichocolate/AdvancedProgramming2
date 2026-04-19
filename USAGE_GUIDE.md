# 📚 Advanced Programming 2 - Микросервисы: Полное Руководство

## 🏗️ Архитектура системы

```
┌──────────────────────────────────────────────────────────────┐
│                     External Client (REST)                   │
└────────────────┬─────────────────────────────────────────────┘
                 │ HTTP REST
                 ↓
         ┌───────────────────┐
         │  Order Service    │ (Port 8086)
         │  - REST API       │
         │  - gRPC Server    │
         └────────┬──────────┘
                  │ gRPC (internal)
                  ↓
         ┌───────────────────┐
         │ Payment Service   │ (Port 8087)
         │  - REST API       │
         │  - gRPC Server    │
         └───────────────────┘

┌──────────────────────────────┐  ┌──────────────────────────────┐
│  PostgreSQL Database         │  │  PostgreSQL Database         │
│  (ap2 schema - orders)       │  │  (postgres schema - payments)│
└──────────────────────────────┘  └──────────────────────────────┘
```

---

## 🚀 Быстрый старт

### 1. Убедитесь, что PostgreSQL запущен

```bash
# Проверка подключения
psql -U biba -d postgres -c "SELECT 1"
```

### 2. Создайте таблицы

```bash
# Orders table
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

# Payments table
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

### 3. Запустите сервисы

**Terminal 1 - Order Service:**
```bash
cd /home/biba/GolandProjects/AdvancedProgramming2
export PAYMENT_GRPC_ADDR=localhost:8088
export ORDER_ADDR=:8086
export ORDER_GRPC_ADDR=:8085
export ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
go run order-service/cmd/order-service/main.go
```

**Terminal 2 - Payment Service:**
```bash
cd /home/biba/GolandProjects/AdvancedProgramming2
export PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
export PAYMENT_ADDR=:8087
export PAYMENT_GRPC_ADDR=:8088
go run payment-service/cmd/payment-service/main.go
```

---

## 📡 Order Service API

### REST Endpoints

#### 1. **Создать заказ** (POST)
```bash
curl -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-001",
    "item_name": "Laptop",
    "amount": 50000
  }'
```

**Ответ (успешно):**
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

**Статусы ответов:**
- `201 Created` — Заказ успешно создан
- `400 Bad Request` — Ошибка в запросе (amount <= 0, отсутствуют поля)
- `503 Service Unavailable` — Payment Service недоступен

**Статусы заказа:**
- `Pending` — Ожидание платежа
- `Paid` — Платёж авторизован (статус "Authorized" от Payment Service)
- `Failed` — Платёж отклонён (статус "Declined" от Payment Service)
- `Cancelled` — Заказ отменён

---

#### 2. **Получить заказ** (GET)
```bash
curl http://localhost:8086/orders/550e8400-e29b-41d4-a716-446655440000
```

**Ответ:**
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

**Статусы ответов:**
- `200 OK` — Заказ найден
- `404 Not Found` — Заказ не существует

---

#### 3. **Отменить заказ** (PATCH)
```bash
curl -X PATCH http://localhost:8086/orders/550e8400-e29b-41d4-a716-446655440000/cancel
```

**Ответ:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "cust-001",
  "item_name": "Laptop",
  "amount": 50000,
  "status": "Cancelled",
  "created_at": "2026-04-19T23:30:00Z"
}
```

**Требования:**
- Заказ должен быть в статусе `Pending` (невозможно отменить уже оплаченный заказ)

**Статусы ответов:**
- `200 OK` — Заказ успешно отменён
- `404 Not Found` — Заказ не существует
- `400 Bad Request` — Заказ не в статусе Pending

---

#### 4. **Получить список заказов** (GET)
```bash
# Все заказы с суммой от 1000 до 50000
curl "http://localhost:8086/orders?min_amount=1000&max_amount=50000"
```

**Параметры:**
- `min_amount` (int64) — Минимальная сумма
- `max_amount` (int64) — Максимальная сумма

**Ответ:**
```json
{
  "orders": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "customer_id": "cust-001",
      "item_name": "Laptop",
      "amount": 50000,
      "status": "Paid",
      "created_at": "2026-04-19T23:30:00Z"
    }
  ]
}
```

---

### gRPC Endpoints

#### SubscribeToOrderUpdates (Server-Side Streaming)

**Proto определение:**
```protobuf
service OrderService {
  rpc SubscribeToOrderUpdates(OrderRequest) 
    returns (stream OrderStatusUpdate);
}

message OrderRequest {
  string order_id = 1;
}

message OrderStatusUpdate {
  string order_id = 1;
  string status = 2;
  string customer_id = 3;
  string item_name = 4;
  int64 amount = 5;
  google.protobuf.Timestamp updated_at = 6;
}
```

**Использование (grpcurl):**
```bash
grpcurl -plaintext \
  -d '{"order_id":"550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:8085 ap2.v1.OrderService/SubscribeToOrderUpdates
```

**Как работает:**
1. Клиент подключается к gRPC серверу Order Service (порт 8085)
2. Отправляет OrderRequest с order_id
3. Получает поток OrderStatusUpdate
4. При изменении статуса заказа в БД, сервер отправляет обновление
5. Соединение остаётся открытым до отмены клиентом или истечения контекста

---

## 💳 Payment Service API

### REST Endpoints

#### 1. **Обработать платёж** (POST)
```bash
curl -X POST http://localhost:8087/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 50000
  }'
```

**Ответ:**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "txn-001-abc123",
  "amount": 50000,
  "status": "Authorized"
}
```

**Правила статуса:**
- `Authorized` — Если amount ≤ 50000
- `Declined` — Если amount > 50000

**Статусы ответов:**
- `201 Created` — Платёж обработан
- `400 Bad Request` — Ошибка (order_id не UUID, amount <= 0)
- `500 Internal Server Error` — Ошибка БД

---

#### 2. **Получить платёж** (GET)
```bash
curl http://localhost:8087/payments/550e8400-e29b-41d4-a716-446655440000
```

**Ответ:**
```json
{
  "id": "660e8400-e29b-41d4-a716-446655440001",
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "transaction_id": "txn-001-abc123",
  "amount": 50000,
  "status": "Authorized"
}
```

**Примечание:** Поиск по order_id (каждый заказ имеет только один платёж)

---

### gRPC Endpoints

#### ProcessPayment (Unary RPC)

**Proto определение:**
```protobuf
service PaymentService {
  rpc ProcessPayment(PaymentRequest) 
    returns (PaymentResponse);
  rpc ListPayments(ListPaymentsRequest)
    returns (ListPaymentsResponse);
}

message PaymentRequest {
  string order_id = 1;
  int64 amount = 2;
}

message PaymentResponse {
  string transaction_id = 1;
  string status = 2;
  google.protobuf.Timestamp processed_at = 3;
}
```

**Использование (grpcurl):**
```bash
grpcurl -plaintext \
  -d '{"order_id":"550e8400-e29b-41d4-a716-446655440000","amount":50000}' \
  localhost:8088 ap2.v1.PaymentService/ProcessPayment
```

---

#### ListPayments (Unary RPC) — Новый метод!

**Proto определение:**
```protobuf
service PaymentService {
  rpc ListPayments(ListPaymentsRequest)
    returns (ListPaymentsResponse);
}

message ListPaymentsRequest {
  string status = 1; // "Authorized" или "Declined"
}

message ListPaymentsResponse {
  repeated PaymentResponse payments = 1;
}
```

**Использование (grpcurl):**
```bash
# Получить все авторизованные платежи
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments

# Получить все отклонённые платежи
grpcurl -plaintext \
  -d '{"status":"Declined"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments
```

**Ответ:**
```json
{
  "payments": [
    {
      "transaction_id": "txn-001-abc123",
      "status": "Authorized",
      "processed_at": "2026-04-19T23:30:00Z"
    }
  ]
}
```

---

## 🔄 Сквозной поток (End-to-End Flow)

### Создание заказа: что происходит внутри

```
1. Client → Order Service (REST POST /orders)
   ↓
2. Order Service REST Handler
   - Парсит JSON
   - Валидирует данные
   ↓
3. Order UseCase (CreateOrder)
   - Генерирует UUID
   - Вычисляет статус = "Pending"
   - Сохраняет в БД
   ↓
4. Order Service → Payment Service (gRPC ProcessPayment)
   - Отправляет order_id и amount
   ↓
5. Payment Service gRPC Handler (ProcessPayment)
   - Валидирует order_id как UUID
   - Определяет статус:
     * "Authorized" если amount ≤ 50000
     * "Declined" если amount > 50000
   - Сохраняет платёж в БД
   - Возвращает ответ
   ↓
6. Order Service получает ответ
   - Если status == "Authorized" → order.status = "Paid"
   - Если status == "Declined" → order.status = "Failed"
   - Обновляет заказ в БД
   - Публикует обновление в broadcaster
   ↓
7. gRPC клиенты (подписчики) получают OrderStatusUpdate
   ↓
8. Order Service возвращает заказ клиенту (REST ответ)
```

### Пример сквозного тестирования

```bash
# Terminal 1: Запустить Order Service
export PAYMENT_GRPC_ADDR=localhost:8088
export ORDER_ADDR=:8086
export ORDER_GRPC_ADDR=:8085
export ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
go run order-service/cmd/order-service/main.go

# Terminal 2: Запустить Payment Service
export PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
export PAYMENT_ADDR=:8087
export PAYMENT_GRPC_ADDR=:8088
go run payment-service/cmd/payment-service/main.go

# Terminal 3: Клиент - подписаться на обновления ДО создания заказа
# (после запуска, она будет ждать обновлений)
grpcurl -plaintext \
  -d '{"order_id":"550e8400-e29b-41d4-a716-446655440000"}' \
  localhost:8085 ap2.v1.OrderService/SubscribeToOrderUpdates

# Terminal 4: Создать заказ
ORDER_ID=$(curl -s -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-001",
    "item_name": "Laptop",
    "amount": 50000
  }' | jq -r '.id')

echo "Created order: $ORDER_ID"

# Terminal 5: Проверить статус
curl http://localhost:8086/orders/$ORDER_ID | jq '.status'

# Terminal 3: Должны увидеть OrderStatusUpdate с новым статусом
```

---

## 🗄️ Структура данных

### Orders Table
```sql
CREATE TABLE orders (
    id UUID PRIMARY KEY,                          -- Уникальный ID заказа
    customer_id TEXT NOT NULL,                    -- ID клиента
    item_name TEXT NOT NULL,                      -- Название товара
    amount BIGINT NOT NULL CHECK (amount > 0),   -- Сумма в копейках/центах
    status TEXT NOT NULL CHECK (
      status IN ('Pending', 'Paid', 'Failed', 'Cancelled')
    ),                                            -- Статус заказа
    created_at TIMESTAMP NOT NULL DEFAULT NOW()  -- Время создания
);
```

**Статусы:**
- `Pending` — Ожидание платежа
- `Paid` — Платёж авторизован
- `Failed` — Платёж отклонён
- `Cancelled` — Отменён пользователем

---

### Payments Table
```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY,                             -- Уникальный ID платежа
    order_id UUID NOT NULL,                          -- ID связанного заказа
    transaction_id TEXT UNIQUE NOT NULL,             -- ID транзакции в системе
    amount BIGINT NOT NULL CHECK (amount > 0),      -- Сумма в копейках/центах
    status TEXT NOT NULL CHECK (
      status IN ('Authorized', 'Declined')
    ),                                               -- Статус авторизации
    created_at TIMESTAMP NOT NULL DEFAULT NOW()     -- Время создания
);
```

**Статусы:**
- `Authorized` — Платёж авторизован (amount ≤ 50000)
- `Declined` — Платёж отклонён (amount > 50000)

---

## ⚙️ Конфигурация

### Environment Variables

```bash
# Order Service
ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
ORDER_ADDR=:8086                          # REST API port
ORDER_GRPC_ADDR=:8085                    # gRPC server port
PAYMENT_GRPC_ADDR=localhost:8088         # Payment Service gRPC адрес

# Payment Service
PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
PAYMENT_ADDR=:8087                       # REST API port
PAYMENT_GRPC_ADDR=:8088                  # gRPC server port
```

### .env файл
Создайте `.env` файл в корне проекта:
```
ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
PAYMENT_SERVICE_URL=http://localhost:8087/payments/
PAYMENT_GRPC_ADDR=localhost:8088
ORDER_ADDR=:8086
ORDER_GRPC_ADDR=:8085
PAYMENT_ADDR=:8087
```

---

## 📊 Примеры использования

### Сценарий 1: Успешный заказ (amount ≤ 50000)

```bash
# 1. Создать заказ
curl -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-123",
    "item_name": "Mouse",
    "amount": 5000
  }'

# Ответ:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "status": "Paid"  ← Автоматически "Paid" (авторизован)
# }

# 2. Проверить платёж
curl http://localhost:8087/payments/550e8400-e29b-41d4-a716-446655440000

# Ответ:
# {
#   "status": "Authorized"  ← Платёж авторизован
# }
```

---

### Сценарий 2: Отклонённый платёж (amount > 50000)

```bash
# 1. Создать заказ с большой суммой
curl -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-456",
    "item_name": "Server",
    "amount": 100000
  }'

# Ответ:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440001",
#   "status": "Failed"  ← Автоматически "Failed" (отклонён)
# }

# 2. Проверить платёж
curl http://localhost:8087/payments/550e8400-e29b-41d4-a716-446655440001

# Ответ:
# {
#   "status": "Declined"  ← Платёж отклонён
# }
```

---

### Сценарий 3: Отмена заказа

```bash
# 1. Создать заказ в статусе Pending
# (Добавить логику паузы перед платежом, или создать заказ в статусе Pending)

# 2. Отменить
curl -X PATCH http://localhost:8086/orders/550e8400-e29b-41d4-a716-446655440000/cancel

# Ответ:
# {
#   "status": "Cancelled"  ← Отменён
# }
```

---

### Сценарий 4: Список платежей по статусу

```bash
# Получить все авторизованные платежи
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments

# Получить все отклонённые платежи
grpcurl -plaintext \
  -d '{"status":"Declined"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments
```

---

## 🐛 Обработка ошибок

### Order Service

| Код | Сообщение | Причина | Решение |
|-----|-----------|---------|---------|
| 400 | `customer_id is required` | Не указан customer_id | Добавьте customer_id в запрос |
| 400 | `amount must be greater than zero` | amount ≤ 0 | Укажите amount > 0 |
| 404 | `order not found` | Заказ не существует | Проверьте order_id |
| 400 | `only Pending orders can be cancelled` | Заказ не в Pending | Отменяются только Pending заказы |
| 503 | `payment service unavailable` | Payment Service недоступен | Убедитесь, что Payment Service запущен на порту 8088 |

### Payment Service

| Код | Сообщение | Причина | Решение |
|-----|-----------|---------|---------|
| 400 | `order_id is required` | Не указан order_id | Добавьте order_id в запрос |
| 400 | `amount is required` | Не указан amount | Добавьте amount в запрос |
| 500 | `order_id must be valid UUID` | order_id не UUID формата | Используйте валидный UUID |
| 404 | `payment not found` | Платёж не найден | Проверьте order_id |

### gRPC ошибки

| Code | Описание |
|------|----------|
| `InvalidArgument` (3) | Невалидные входные параметры |
| `Internal` (13) | Внутренняя ошибка сервера |
| `Unimplemented` (12) | Метод не реализован |
| `Unavailable` (14) | Сервис недоступен |

---

## 🧪 Тестирование

### Инструменты

```bash
# REST testing
curl

# gRPC testing
grpcurl

# JSON processing
jq
```

### Установка grpcurl

```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Полный тестовый скрипт

```bash
#!/bin/bash

# Цвета для вывода
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Order Service Tests ===${NC}"

# Test 1: Create order
echo -e "${BLUE}1. Creating order...${NC}"
ORDER=$(curl -s -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "test-user",
    "item_name": "Keyboard",
    "amount": 15000
  }')

ORDER_ID=$(echo $ORDER | jq -r '.id')
ORDER_STATUS=$(echo $ORDER | jq -r '.status')

echo -e "${GREEN}Created order: $ORDER_ID with status: $ORDER_STATUS${NC}"

# Test 2: Get order
echo -e "${BLUE}2. Getting order...${NC}"
curl -s http://localhost:8086/orders/$ORDER_ID | jq '.'

# Test 3: List payments
echo -e "${BLUE}3. Listing authorized payments...${NC}"
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments | jq '.'

echo -e "${GREEN}All tests completed!${NC}"
```

---

## 📁 Структура проекта

```
.
├── order-service/
│   ├── cmd/order-service/
│   │   └── main.go                    # Entry point
│   ├── internal/
│   │   ├── app/
│   │   │   └── app.go                 # Dependency injection
│   │   ├── domain/
│   │   │   └── order.go               # Order entity
│   │   ├── repository/
│   │   │   └── order_postgres.go      # Database queries
│   │   ├── transport/
│   │   │   ├── grpc_server.go         # gRPC handler
│   │   │   ├── order_handler.go       # REST handler
│   │   │   └── payment_grpc_client.go # Payment client
│   │   └── usecase/
│   │       ├── order_usecase.go       # Business logic
│   │       ├── broadcaster.go         # Event broadcaster
│   │       └── ports.go               # Interfaces
│   └── migrations/
│       └── 001_create_orders.sql      # Schema
│
├── payment-service/
│   ├── cmd/payment-service/
│   │   └── main.go                    # Entry point
│   ├── internal/
│   │   ├── app/
│   │   │   └── app.go                 # Dependency injection
│   │   ├── domain/
│   │   │   └── payment.go             # Payment entity
│   │   ├── repository/
│   │   │   └── payment_postgres.go    # Database queries
│   │   ├── transport/
│   │   │   ├── grpc_server.go         # gRPC handler
│   │   │   └── payment_handler.go     # REST handler
│   │   └── usecase/
│   │       ├── payment_usecase.go     # Business logic
│   │       └── ports.go               # Interfaces
│   └── migrations/
│       └── 001_create_payments.sql    # Schema
│
├── .env                               # Configuration
├── go.mod                             # Go modules
├── go.sum                             # Checksums
├── README.md                          # Project overview
└── USAGE_GUIDE.md                     # This file
```

---

## 🔗 Полезные ссылки

- **Proto Repository**: https://github.com/Gorkyichocolate/Proto
- **Generated Code**: https://github.com/Gorkyichocolate/ap2-generated
- **grpcurl docs**: https://github.com/fullstorydev/grpcurl
- **Protocol Buffers**: https://developers.google.com/protocol-buffers

---

## ❓ Часто задаваемые вопросы

### Q: Как подключиться к Order Service через gRPC?
**A:** Используйте адрес `localhost:8085` (переменная `ORDER_GRPC_ADDR`). Убедитесь, что заказ запущен.

### Q: Как изменить лимит платежа (сейчас 50000)?
**A:** Отредактируйте `payment-service/internal/domain/payment.go`, константа `PaymentLimit`.

### Q: Что если Payment Service недоступен?
**A:** Order Service вернёт 503 Service Unavailable. Убедитесь, что Payment Service запущен на `PAYMENT_GRPC_ADDR`.

### Q: Можно ли запустить на разных машинах?
**A:** Да. Просто установите `PAYMENT_GRPC_ADDR` на IP-адрес Payment Service машины.

### Q: Как подписаться на обновления заказа в реальном времени?
**A:** Используйте `SubscribeToOrderUpdates` gRPC streaming endpoint.

---

## 📝 Чек-лист для новых разработчиков

- [ ] PostgreSQL установлен и запущен
- [ ] Созданы таблицы orders и payments
- [ ] Переменные окружения установлены
- [ ] Запущены оба сервиса
- [ ] Успешно создан первый заказ через REST API
- [ ] Получен список платежей через gRPC
- [ ] Подписан на обновления заказа через gRPC streaming

---

**Версия документации**: 1.0  
**Последнее обновление**: 2026-04-19  
**Статус**: ✅ Production Ready
