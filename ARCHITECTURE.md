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
