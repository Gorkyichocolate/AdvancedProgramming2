# 📖 Advanced Programming 2 - Документация

Добро пожаловать! Выберите нужный раздел документации:

---

## 🚀 Начинаем

### Для новых разработчиков
1. **Прочитайте**: [README.md](README.md) — Общий обзор проекта
2. **Запустите**: [QUICK_REFERENCE.md](QUICK_REFERENCE.md) — Быстрый старт с командами
3. **Изучите**: [ARCHITECTURE.md](ARCHITECTURE.md) — Диаграммы и архитектура

### Для детального понимания
1. **Полный гайд**: [USAGE_GUIDE.md](USAGE_GUIDE.md) — Все о микросервисах
2. **Примеры**: [DEMO_LIST_PAYMENTS.md](DEMO_LIST_PAYMENTS.md) — Практический пример

---

## 📚 Все документы

| Документ | Назначение | Для кого |
|----------|-----------|----------|
| [README.md](README.md) | Описание проекта и архитектуры | Все |
| [QUICK_REFERENCE.md](QUICK_REFERENCE.md) | Шпаргалка с командами | Разработчики |
| [USAGE_GUIDE.md](USAGE_GUIDE.md) | Полное руководство по использованию | Разработчики |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Диаграммы и архитектурные паттерны | Архитекторы, Разработчики |
| [DEMO_LIST_PAYMENTS.md](DEMO_LIST_PAYMENTS.md) | Пример реализации ListPayments RPC | Разработчики |
| [IMPLEMENTATION_REPORT.md](IMPLEMENTATION_REPORT.md) | Отчёт о реализации ListPayments | Разработчики |

---

## 🎯 Что мне нужно?

### ✅ Я хочу быстро запустить сервисы
→ [QUICK_REFERENCE.md](QUICK_REFERENCE.md#-быстрый-старт)

### ✅ Я хочу понять архитектуру
→ [ARCHITECTURE.md](ARCHITECTURE.md)

### ✅ Я хочу использовать REST API
→ [USAGE_GUIDE.md](USAGE_GUIDE.md#-order-service-api) и [USAGE_GUIDE.md](USAGE_GUIDE.md#-payment-service-api)

### ✅ Я хочу использовать gRPC
→ [USAGE_GUIDE.md](USAGE_GUIDE.md#-order-service-api) (SubscribeToOrderUpdates) и [USAGE_GUIDE.md](USAGE_GUIDE.md#-payment-service-api) (ProcessPayment, ListPayments)

### ✅ Мне нужны примеры кода
→ [USAGE_GUIDE.md](USAGE_GUIDE.md#-примеры-использования)

### ✅ Я хочу протестировать сервисы
→ [QUICK_REFERENCE.md](QUICK_REFERENCE.md#-complete-test-flow) или [USAGE_GUIDE.md](USAGE_GUIDE.md#-тестирование)

### ✅ Я хочу разобраться с новым ListPayments RPC
→ [DEMO_LIST_PAYMENTS.md](DEMO_LIST_PAYMENTS.md) → [IMPLEMENTATION_REPORT.md](IMPLEMENTATION_REPORT.md)

---

## 📡 API Endpoints

### Order Service (REST)
```
POST   http://localhost:8086/orders                    # Create order
GET    http://localhost:8086/orders/:id               # Get order
PATCH  http://localhost:8086/orders/:id/cancel        # Cancel order
GET    http://localhost:8086/orders?min_amount=1000   # List orders
```

### Order Service (gRPC)
```
localhost:8085 - ap2.v1.OrderService/SubscribeToOrderUpdates
```

### Payment Service (REST)
```
POST   http://localhost:8087/payments                 # Process payment
GET    http://localhost:8087/payments/:order_id       # Get payment
```

### Payment Service (gRPC)
```
localhost:8088 - ap2.v1.PaymentService/ProcessPayment
localhost:8088 - ap2.v1.PaymentService/ListPayments
```

---

## 🗄️ Структура БД

### Orders Table (ap2 schema)
```sql
id (UUID) | customer_id | item_name | amount | status | created_at
```
**Статусы**: Pending, Paid, Failed, Cancelled

### Payments Table (postgres schema)
```sql
id (UUID) | order_id | transaction_id | amount | status | created_at
```
**Статусы**: Authorized, Declined

---

## 🔑 Ключевые концепции

### Bounded Contexts
- **Order Service** — управление жизненным циклом заказа
- **Payment Service** — обработка платежей

### Communication
- **REST** — клиент → Order Service (внешний API)
- **gRPC** — Order Service → Payment Service (внутренний)
- **gRPC Streaming** — Order Service → Client (подписка на события)

### Database Per Service
- Orders: PostgreSQL (ap2 schema)
- Payments: PostgreSQL (postgres schema)

### Contract-First
- Proto контракт: `/home/biba/GolandProjects/Proto`
- Generated код: `github.com/Gorkyichocolate/ap2-generated`

---

## 📊 Примеры

### Создать и получить заказ
```bash
# 1. Создать
ORDER_ID=$(curl -s -X POST http://localhost:8086/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-1",
    "item_name": "Laptop",
    "amount": 50000
  }' | jq -r '.id')

# 2. Получить
curl http://localhost:8086/orders/$ORDER_ID | jq '.'
```

### Получить платежи по статусу
```bash
grpcurl -plaintext \
  -d '{"status":"Authorized"}' \
  localhost:8088 ap2.v1.PaymentService/ListPayments
```

### Подписаться на обновления заказа
```bash
grpcurl -plaintext \
  -d "{\"order_id\":\"$ORDER_ID\"}" \
  localhost:8085 ap2.v1.OrderService/SubscribeToOrderUpdates
```

→ Больше примеров в [USAGE_GUIDE.md](USAGE_GUIDE.md#-примеры-использования)

---

## 🔧 Окружение

```bash
# Order Service
ORDER_DB_URL=postgres://biba:123456@localhost:5432/ap2
ORDER_ADDR=:8086
ORDER_GRPC_ADDR=:8085
PAYMENT_GRPC_ADDR=localhost:8088

# Payment Service
PAYMENT_DB_URL=postgres://biba:123456@localhost:5432/postgres
PAYMENT_ADDR=:8087
PAYMENT_GRPC_ADDR=:8088
```

---

## 🚦 Статусы ответов

| Код | Значение | Примеры |
|-----|----------|---------|
| 200 | OK | Успешная GET, успешная PATCH |
| 201 | Created | Успешная POST (create) |
| 400 | Bad Request | Невалидные параметры, amount <= 0 |
| 404 | Not Found | Заказ/платёж не существует |
| 500 | Internal Error | Ошибка БД |
| 503 | Service Unavailable | Payment Service недоступен |

---

## ❓ FAQ

**Q: На каком порту Order Service?**  
A: REST - 8086, gRPC - 8085

**Q: На каком порту Payment Service?**  
A: REST - 8087, gRPC - 8088

**Q: Как менять лимит платежа (сейчас 50000)?**  
A: `payment-service/internal/domain/payment.go` - константа `PaymentLimit`

**Q: Что если Payment Service недоступен?**  
A: Order Service вернёт 503 Service Unavailable

**Q: Как запустить на разных машинах?**  
A: Установите `PAYMENT_GRPC_ADDR` на IP-адрес Payment Service

**Q: Как подписаться на обновления в реальном времени?**  
A: Используйте gRPC streaming endpoint `SubscribeToOrderUpdates`

---

## 📞 Полезные команды

```bash
# Проверить, запущены ли сервисы
curl http://localhost:8086/orders 2>/dev/null && echo "Order Service OK"
curl http://localhost:8087/payments 2>/dev/null && echo "Payment Service OK"

# Проверить логи
tail -f /tmp/order-service.log
tail -f /tmp/payment-service.log

# Тестировать gRPC доступность
grpcurl -plaintext localhost:8085 list
grpcurl -plaintext localhost:8088 list

# Проверить БД
psql -U biba -d ap2 -c "SELECT COUNT(*) FROM orders;"
psql -U biba -d postgres -c "SELECT COUNT(*) FROM payments;"
```

---

## 🎓 Обучающие материалы

- Protocol Buffers: https://developers.google.com/protocol-buffers
- gRPC: https://grpc.io/docs/
- Hexagonal Architecture: https://alistair.cockburn.us/hexagonal-architecture/
- Domain-Driven Design: https://www.domainlanguage.com/ddd/

---

## 📝 История документации

| Версия | Дата | Описание |
|--------|------|----------|
| 1.0 | 2026-04-19 | Начальная версия: USAGE_GUIDE + QUICK_REFERENCE + ARCHITECTURE |
| | | Добавлена ListPayments RPC |

---

**Последнее обновление**: 2026-04-19  
**Статус**: ✅ Production Ready  
**Версия**: 1.0

🚀 Готовы начать? → [QUICK_REFERENCE.md](QUICK_REFERENCE.md#-быстрый-старт)
