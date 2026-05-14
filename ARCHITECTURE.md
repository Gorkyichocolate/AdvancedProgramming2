# 🏗️ Architecture Diagrams

## System Architecture

```mermaid
graph TB
    Client["🖥️ External Client"]
    
    subgraph OrderService ["Order Service (8086/8085)"]
        OrderREST["🌐 REST Handler<br/>/orders"]
        OrderUC["📋 Order UseCase<br/>CreateOrder<br/>GetOrder<br/>CancelOrder"]
        OrderRepo["💾 Order Repository<br/>PostgreSQL"]
        OrderGRPC["🔌 gRPC Server<br/>SubscribeToOrderUpdates"]
    end
    
    subgraph PaymentService ["Payment Service (8087/8088)"]
        PaymentREST["🌐 REST Handler<br/>/payments"]
        PaymentUC["💳 Payment UseCase<br/>ProcessPayment<br/>ListPaymentsByStatus"]
        PaymentRepo["💾 Payment Repository<br/>PostgreSQL"]
        PaymentGRPC["🔌 gRPC Server<br/>ProcessPayment<br/>ListPayments"]
    end
    
    DB_Orders["🗄️ PostgreSQL<br/>ap2 schema<br/>orders table"]
    DB_Payments["🗄️ PostgreSQL<br/>postgres schema<br/>payments table"]
    
    Client -->|REST POST /orders| OrderREST
    Client -->|REST GET /orders| OrderREST
    OrderREST --> OrderUC
    OrderUC --> OrderRepo
    OrderRepo --> DB_Orders
    
    OrderUC -->|gRPC ProcessPayment| PaymentGRPC
    PaymentGRPC --> PaymentUC
    PaymentUC --> PaymentRepo
    PaymentRepo --> DB_Payments
    
    OrderUC -->|Broadcaster| OrderGRPC
    Client -->|gRPC Subscribe| OrderGRPC
    
    style Client fill:#4A90E2
    style OrderService fill:#7ED321
    style PaymentService fill:#F5A623
    style DB_Orders fill:#50E3C2
    style DB_Payments fill:#50E3C2
```

---

## Create Order Flow (Sequence Diagram)

```mermaid
sequenceDiagram
    participant Client as 🖥️ Client
    participant OrderAPI as Order REST API
    participant OrderUC as Order UseCase
    participant OrderDB as Order DB
    participant PaymentGRPC as Payment gRPC
    participant PaymentUC as Payment UseCase
    participant PaymentDB as Payment DB
    
    Client->>OrderAPI: POST /orders<br/>{customer_id, item_name, amount}
    OrderAPI->>OrderUC: CreateOrder(ctx, ...)
    
    OrderUC->>OrderUC: Validate amount > 0
    OrderUC->>OrderDB: Create order in Pending state
    
    OrderUC->>PaymentGRPC: ProcessPayment(order_id, amount)
    
    PaymentGRPC->>PaymentUC: ProcessPayment(order_id, amount)
    PaymentUC->>PaymentUC: Determine status<br/>if amount ≤ 50000: Authorized<br/>else: Declined
    PaymentUC->>PaymentDB: Create payment record
    PaymentUC->>PaymentGRPC: Return PaymentResponse
    
    PaymentGRPC->>OrderUC: Return {status, transaction_id}
    
    OrderUC->>OrderUC: Map payment status to order status<br/>Authorized → Paid<br/>Declined → Failed
    OrderUC->>OrderDB: Update order status
    OrderUC->>OrderUC: Publish OrderStatusUpdate<br/>to broadcaster
    
    OrderUC->>OrderAPI: Return order
    OrderAPI->>Client: 201 Created {order with Paid/Failed status}
```

---

## Subscribe to Order Updates (gRPC Streaming)

```mermaid
sequenceDiagram
    participant Client as 🖥️ gRPC Client
    participant OrderGRPC as Order gRPC Server
    participant Broadcaster as Event Broadcaster
    participant OrderDB as Order DB
    
    Client->>OrderGRPC: SubscribeToOrderUpdates(order_id)
    OrderGRPC->>OrderDB: GetOrder(order_id)
    OrderGRPC->>Client: Send current OrderStatusUpdate
    
    OrderGRPC->>Broadcaster: Subscribe(ctx, order_id)
    
    Note over OrderDB,Broadcaster: Wait for changes...
    
    Note over OrderDB,Broadcaster: Order status changes (e.g., via API)
    OrderDB->>Broadcaster: Publish updated order
    Broadcaster->>OrderGRPC: Receive filtered update (same order_id)
    OrderGRPC->>Client: Send OrderStatusUpdate
    
    Client->>Client: Receive stream update in real-time
    
    Note over OrderDB,Broadcaster: Connection continues until:<br/>Client cancels<br/>Context timeout<br/>Server shuts down
```

---

## Component Interaction Diagram

```mermaid
graph LR
    subgraph Clients
        REST["🌐 REST Client"]
        GRPC_Client["🔌 gRPC Client"]
    end
    
    subgraph OrderService_Internal
        OrderHandler["Order Handler"]
        OrderUseCase["Order UseCase"]
        OrderRepository["Order Repository"]
        OrderBroadcaster["Broadcaster"]
        GRPCServer_Order["gRPC Server"]
    end
    
    subgraph PaymentService_Internal
        PaymentClient["Payment gRPC Client"]
        PaymentHandler["Payment Handler"]
        PaymentUseCase["Payment UseCase"]
        PaymentRepository["Payment Repository"]
        GRPCServer_Payment["gRPC Server"]
    end
    
    subgraph Databases
        OrderDB["Orders DB"]
        PaymentDB["Payments DB"]
    end
    
    REST -->|"REST /orders"| OrderHandler
    OrderHandler -->|call| OrderUseCase
    OrderUseCase -->|"gRPC call"| PaymentClient
    
    PaymentClient -->|gRPC invoke| GRPCServer_Payment
    GRPCServer_Payment -->|call| PaymentUseCase
    PaymentUseCase -->|query| PaymentRepository
    PaymentRepository -->|SQL| PaymentDB
    
    OrderUseCase -->|query| OrderRepository
    OrderRepository -->|SQL| OrderDB
    
    OrderUseCase -->|publish| OrderBroadcaster
    OrderBroadcaster -->|filter & send| GRPCServer_Order
    GRPCServer_Order -->|stream| GRPC_Client
```

---

## Data Flow for Different HTTP Methods

### POST /orders (Create Order)

```mermaid
graph LR
    A["Request Body<br/>{customer_id,<br/>item_name,<br/>amount}"]
    B["Validate<br/>amount > 0"]
    C["Create order<br/>status=Pending"]
    D["Call Payment Service<br/>gRPC"]
    E["Update order status<br/>based on payment"]
    F["Return 201<br/>order"]
    
    A --> B --> C --> D --> E --> F
```

### GET /orders (List Orders)

```mermaid
graph LR
    A["Query String<br/>min_amount,<br/>max_amount"]
    B["Validate<br/>ranges"]
    C["Query DB<br/>WHERE amount BETWEEN"]
    D["Return 200<br/>orders"]
    
    A --> B --> C --> D
```

### PATCH /orders/:id/cancel (Cancel Order)

```mermaid
graph LR
    A["Path param<br/>order_id"]
    B["Get order<br/>from DB"]
    C["Check status<br/>== Pending?"]
    D["Update status<br/>to Cancelled"]
    E["Publish event<br/>to Broadcaster"]
    F["Return 200<br/>or 400"]
    
    A --> B --> C --> D --> E --> F
```

---

## gRPC Service Definition

```mermaid
graph TB
    subgraph PaymentService
        ProcessPayment["ProcessPayment<br/>Request: order_id, amount<br/>Response: transaction_id, status"]
        ListPayments["ListPayments<br/>Request: status<br/>Response: repeated PaymentResponse"]
    end
    
    subgraph OrderService
        SubscribeToOrderUpdates["SubscribeToOrderUpdates<br/>Request: order_id<br/>Response: stream OrderStatusUpdate"]
    end
    
    Client["🖥️ Client<br/>(gRPC)"]
    
    Client -->|Unary| ProcessPayment
    Client -->|Unary| ListPayments
    Client -->|Streaming| SubscribeToOrderUpdates
    
    style ProcessPayment fill:#F5A623
    style ListPayments fill:#F5A623
    style SubscribeToOrderUpdates fill:#7ED321
```

---

## Error Handling Flow

```mermaid
graph TD
    Request["Request received"]
    
    Validate["Validate input"]
    ValidOK{Input OK?}
    
    ValidOK -->|No| BadRequest["Return 400<br/>InvalidArgument"]
    ValidOK -->|Yes| Process["Process request"]
    
    Process --> Execute{Execution<br/>successful?}
    
    Execute -->|DB Error| InternalError["Return 500<br/>Internal"]
    Execute -->|Service Unavailable| Unavailable["Return 503<br/>Unavailable"]
    Execute -->|Yes| Success["Return 200/201<br/>Success"]
    
    BadRequest --> Client["Response to Client"]
    InternalError --> Client
    Unavailable --> Client
    Success --> Client
    
    style BadRequest fill:#E24A4A
    style InternalError fill:#E24A4A
    style Unavailable fill:#E24A4A
    style Success fill:#7ED321
```

---

## Database Schema Relationships

```mermaid
erDiagram
    ORDERS ||--|| PAYMENTS : has
    
    ORDERS {
        uuid id PK
        string customer_id
        string item_name
        bigint amount
        string status
        timestamp created_at
    }
    
    PAYMENTS {
        uuid id PK
        uuid order_id FK
        string transaction_id "UNIQUE"
        bigint amount
        string status
        timestamp created_at
    }
    
    NOTE "One order can have ONE payment"
    NOTE "Payment status determines order status"
```

---

## Deployment Architecture

```mermaid
graph TB
    subgraph Docker_Container_1 ["Docker Container: Order Service"]
        OApp["Order Service App"]
        OEnv["Environment:<br/>ORDER_ADDR=:8086<br/>ORDER_GRPC_ADDR=:8085<br/>PAYMENT_GRPC_ADDR=..."]
    end
    
    subgraph Docker_Container_2 ["Docker Container: Payment Service"]
        PApp["Payment Service App"]
        PEnv["Environment:<br/>PAYMENT_ADDR=:8087<br/>PAYMENT_GRPC_ADDR=:8088"]
    end
    
    subgraph Networking ["Container Network"]
        Network["Docker Bridge Network"]
    end
    
    subgraph Persistence ["Data Persistence"]
        PG["PostgreSQL Container"]
    end
    
    OApp --> Network
    PApp --> Network
    Network --> PG
    
    style Docker_Container_1 fill:#7ED321
    style Docker_Container_2 fill:#F5A623
    style Persistence fill:#50E3C2
```

---

# 📊 Lectures 7-9: Redis Caching, Provider Adapter & Background Jobs

## Architecture with Redis & Notification Service

```mermaid
graph TB
    Client["🖥️ Client"]
    
    subgraph OrderService ["Order Service"]
        OrderHandler["Handler<br/>GET /orders/:id"]
        OrderCache["Cache Layer<br/>Redis"]
        OrderDB["PostgreSQL<br/>orders table"]
    end
    
    subgraph PaymentService ["Payment Service"]
        PaymentAPI["REST API<br/>POST /payments"]
        PaymentDB["PostgreSQL<br/>payments table"]
        RabbitMQ["RabbitMQ<br/>payment.completed"]
    end
    
    subgraph NotificationService ["Notification Service"]
        Consumer["Message Consumer<br/>RabbitMQ"]
        JobProcessor["Job Processor<br/>Retry + Backoff"]
        Provider["Provider Interface"]
        Redis["Redis<br/>Job Status"]
        Simulated["Simulated Provider<br/>Latency + Failures"]
        SMTP["SMTP Provider<br/>Real Email"]
    end
    
    Client -->|GET /orders/:id| OrderHandler
    OrderHandler -->|Check Cache| OrderCache
    OrderCache -->|Miss| OrderDB
    OrderDB -->|Data| OrderCache
    OrderCache -->|Hit| OrderHandler
    OrderHandler -->|Response| Client
    
    Client -->|POST /payments| PaymentAPI
    PaymentAPI -->|Save| PaymentDB
    PaymentAPI -->|Publish Event| RabbitMQ
    
    RabbitMQ -->|Consume| Consumer
    Consumer -->|Process Job| JobProcessor
    JobProcessor -->|Check Status| Redis
    JobProcessor -->|Send via| Provider
    Provider -->|Use| Simulated
    Provider -->|Use| SMTP
    JobProcessor -->|Update Status| Redis
    
    Simulated -->|Simulate| Email["📧 Email<br/>Simulated"]
    SMTP -->|Send| Email
    
    style OrderService fill:#7ED321
    style PaymentService fill:#F5A623
    style NotificationService fill:#FF6B6B
    style OrderCache fill:#50E3C2
    style Redis fill:#50E3C2
```

---

## Cache-Aside Pattern: GET Order

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Cache as Redis Cache
    participant DB as PostgreSQL
    
    Client->>Handler: GET /orders/123
    Handler->>Cache: Get order:123
    
    alt Cache Hit
        Cache-->>Handler: Order data
        Handler-->>Client: 200 OK (cached)
    else Cache Miss
        Cache-->>Handler: nil
        Handler->>DB: SELECT * FROM orders WHERE id=123
        DB-->>Handler: Order data
        Handler->>Cache: SET order:123 + TTL(300s)
        Handler-->>Client: 200 OK (from DB)
    end
```

---

## Cache Invalidation on Status Change

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant UseCase
    participant DB as PostgreSQL
    participant Cache as Redis
    
    Client->>Handler: PATCH /orders/123/cancel
    Handler->>UseCase: CancelOrder(123)
    UseCase->>DB: UPDATE orders SET status='cancelled'
    DB-->>UseCase: OK
    UseCase-->>Handler: Updated order
    Handler->>Cache: DEL order:123
    Cache-->>Handler: OK
    Handler-->>Client: 200 OK
```

---

## Background Job Processing with Retry & Idempotency

```mermaid
sequenceDiagram
    participant RabbitMQ
    participant Consumer
    participant JobProcessor
    participant Redis as Redis<br/>Job Status
    participant Provider
    participant Email
    
    RabbitMQ->>Consumer: Consume payment.completed
    Consumer->>Consumer: Parse PaymentEvent
    Consumer->>JobProcessor: ProcessJob(job)
    
    JobProcessor->>Redis: GET job:notification:{payment_id}
    
    alt Idempotent: Already Processed
        Redis-->>JobProcessor: {"status":"success"}
        JobProcessor-->>Consumer: nil (skip)
        Consumer->>RabbitMQ: ACK message
    else New Job
        Redis-->>JobProcessor: nil
        JobProcessor->>Provider: SendNotification()
        
        alt Success
            Provider->>Email: Send email
            Email-->>Provider: OK
            Provider-->>JobProcessor: nil
            JobProcessor->>Redis: SET status=success, TTL=24h
            JobProcessor-->>Consumer: nil
            Consumer->>RabbitMQ: ACK message
        else Failure
            Provider-->>JobProcessor: error
            JobProcessor->>Redis: SET status=pending,<br/>NextRetry=now+backoff
            JobProcessor-->>Consumer: error
            Consumer->>RabbitMQ: NACK message (requeue)
            Note over RabbitMQ: Message back to queue<br/>for retry with backoff
        end
    end
```

---

## Exponential Backoff Timeline

```
Attempt 1 (Time: 0s)
    │ FAIL
    ├─ Backoff: 2000ms * 2^0 = 2s
    └─ NextRetry: now + 2s
          │
          └──────────────────────▶ Wait 2 seconds

Attempt 2 (Time: +2s)
    │ FAIL
    ├─ Backoff: 2000ms * 2^1 = 4s
    └─ NextRetry: now + 4s
          │
          └──────────────────────────────────▶ Wait 4 seconds

Attempt 3 (Time: +6s)
    │ FAIL
    ├─ Backoff: 2000ms * 2^2 = 8s
    └─ NextRetry: now + 8s
          │
          └──────────────────────────────────────────────────▶ Wait 8 seconds

Attempt 4 (Time: +14s)
    │ SUCCESS ✓
    └─ Job completed, stop retrying
```

---

## Provider Adapter Pattern

```mermaid
graph TB
    App["Notification Service<br/>Job Processor"]
    
    Interface["NotificationProvider<br/>Interface<br/><br/>SendNotification()"]
    
    SimulatedImpl["SimulatedProvider<br/>• Configurable latency<br/>• Random failures<br/>• Log output<br/>• For testing"]
    
    SMTPImpl["SMTPProvider<br/>• Real SMTP server<br/>• Actual emails<br/>• Production ready<br/>• Error handling"]
    
    Factory["Provider Factory<br/>PROVIDER_MODE env<br/>→ Create implementation"]
    
    App -->|Depends on| Interface
    Interface ---|implements| SimulatedImpl
    Interface ---|implements| SMTPImpl
    Factory -->|Creates| SimulatedImpl
    Factory -->|Creates| SMTPImpl
    App -->|Calls| Factory
    
    style Interface fill:#FFD700
    style SimulatedImpl fill:#87CEEB
    style SMTPImpl fill:#90EE90
    style Factory fill:#FFB6C1
```

---

## Data Storage: Redis Key Patterns

```
Job Status (Idempotency):
┌─────────────────────────────────────────┐
│ Key: job:notification:{payment_id}      │
│ TTL: 24h (success), 7d (failed)         │
│                                          │
│ Value:                                   │
│ {                                        │
│   "status": "success|pending|failed",   │
│   "attempt_count": 1,                    │
│   "last_attempt": "2024-05-14T...",      │
│   "next_retry": "2024-05-14T...",        │
│   "error": "simulated error"             │
│ }                                        │
└─────────────────────────────────────────┘

Order Cache (Cache-Aside):
┌─────────────────────────────────────────┐
│ Key: order:{order_id}                   │
│ TTL: 300s (configurable)                │
│                                          │
│ Value:                                   │
│ {                                        │
│   "id": "order-123",                     │
│   "customer_id": "cust-456",             │
│   "item_name": "Widget",                 │
│   "amount": 1000,                        │
│   "status": "PENDING|PAID|CANCELLED"     │
│ }                                        │
└─────────────────────────────────────────┘
```

---

## Component Interactions Summary

| Feature | Component | Lecture | Pattern |
|---------|-----------|---------|---------|
| Order caching | Redis + Order Service | 7 | Cache-aside |
| Cache invalidation | Handler + Redis | 7 | Atomic updates |
| Notification provider | NotificationProvider interface | 8 | Adapter |
| Provider selection | Factory + Env config | 8 | Factory |
| Job processing | JobProcessor + RabbitMQ | 8-9 | Background worker |
| Idempotency | Redis job status | 8-9 | Idempotency record |
| Retry logic | JobProcessor | 8-9 | Retry pattern |
| Exponential backoff | JobProcessor | 8-9 | Backoff strategy |

---

## Configuration Layer

```yaml
Environment Variables:
├── REDIS_URL: redis://localhost:6379
├── CACHE_TTL_SECONDS: 300
├── PROVIDER_MODE: SIMULATED|REAL
├── SIMULATED_FAILURE_RATE: 0.2
├── SIMULATED_LATENCY_MS: 500
├── PROVIDER_RETRY_MAX_ATTEMPTS: 5
├── PROVIDER_RETRY_INITIAL_BACKOFF_MS: 2000
├── PROVIDER_RETRY_MAX_BACKOFF_MS: 32000
└── SMTP_*: For REAL provider mode
```

All behavior controlled by environment variables - no code changes needed!
