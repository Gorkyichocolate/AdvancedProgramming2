# Quick Start Guide: Redis Caching, Provider Adapter, and Background Jobs

## One-Command Setup

```bash
# From project root
docker-compose up --build
```

This starts:
- PostgreSQL (port 5432)
- Redis (port 6379)
- RabbitMQ (port 5672, management UI at 15672)
- Order Service (HTTP: 8001, gRPC: 9001)
- Payment Service (HTTP: 8002, gRPC: 9002)
- Notification Service (listening to RabbitMQ)

## Test the Cache-Aside Pattern

### 1. Create an Order
```bash
ORDER_ID=$(curl -s -X POST http://localhost:8001/orders \
  -H "Content-Type: application/json" \
  -d '{
    "customer_id": "cust-456",
    "item_name": "Laptop",
    "amount": 120000
  }' | jq -r '.id')

echo "Created order: $ORDER_ID"
```

### 2. First Read (Cache Miss)
```bash
# Watch logs for "cache miss" message
docker-compose logs -f order-service &

curl http://localhost:8001/orders/$ORDER_ID

# Logs will show: cache get error or cache miss
```

### 3. Second Read (Cache Hit)
```bash
curl http://localhost:8001/orders/$ORDER_ID

# Logs will show: "cache hit for order..."
# Response time should be <10ms vs ~50ms for DB
```

### 4. Verify Cache in Redis
```bash
docker exec $(docker-compose ps -q redis) redis-cli GET "order:$ORDER_ID"
```

### 5. Invalidate Cache
```bash
# Cancel order - should invalidate cache
curl -X PATCH http://localhost:8001/orders/$ORDER_ID/cancel

# Next read will query database again
curl http://localhost:8001/orders/$ORDER_ID
```

## Test the Notification Provider

### Monitor Notifications (Terminal 1)
```bash
docker-compose logs -f notification-service
```

### Trigger a Payment (Terminal 2)
```bash
curl -X POST http://localhost:8002/payments \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "'$ORDER_ID'",
    "amount": 120000,
    "customer_email": "customer@example.com"
  }'
```

### Watch the Output

**SIMULATED mode (default):**
```
[SIMULATED] Simulating network latency: 500ms
[SIMULATED] Notification sent to: customer@example.com
[SIMULATED] Subject: Payment Confirmation
Job completed successfully: {order_id} (attempt 1)
```

**With simulated failures (~20% failure rate):**
```
[SIMULATED] ... random failure (probability: 20.00%)
Job failed (attempt 1/5): {order_id}. Next retry at [time] with backoff 2000ms
[RETRYING after 2s delay]
[SIMULATED] Notification sent to: customer@example.com
Job completed successfully: {order_id} (attempt 2)
```

## Test Idempotency

### Method 1: Duplicate Message Simulation

```bash
# Send same payment twice
PAYMENT_ID="payment-123"

curl -X POST http://localhost:8002/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $PAYMENT_ID" \
  -d '{"order_id":"'$ORDER_ID'","amount":100,"customer_email":"test@example.com"}'

# Send again with same ID
curl -X POST http://localhost:8002/payments \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: $PAYMENT_ID" \
  -d '{"order_id":"'$ORDER_ID'","amount":100,"customer_email":"test@example.com"}'

# Watch logs - only one notification should be sent
docker-compose logs notification-service | grep "Job completed successfully"
```

### Method 2: Check Job Status in Redis

```bash
# After sending a payment, check job status
docker exec $(docker-compose ps -q redis) redis-cli GET "job:notification:$ORDER_ID"

# Output:
# {"status":"success","last_attempt":"2024-05-14T...","attempt_count":1}
```

## Test Exponential Backoff

### Enable Higher Failure Rate for Testing

1. Edit `.env`:
```env
SIMULATED_FAILURE_RATE=0.8    # 80% failure rate for testing
SIMULATED_LATENCY_MS=200
PROVIDER_RETRY_MAX_ATTEMPTS=5
PROVIDER_RETRY_INITIAL_BACKOFF_MS=1000
PROVIDER_RETRY_MAX_BACKOFF_MS=16000
```

2. Restart services:
```bash
docker-compose down
docker-compose up --build
```

3. Send payment:
```bash
curl -X POST http://localhost:8002/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":"test-123","amount":100,"customer_email":"test@example.com"}'
```

4. Watch exponential backoff in action:
```bash
docker-compose logs -f notification-service | grep "backoff\|attempt"
```

Expected output:
```
Job failed (attempt 1/5): ... backoff 1000ms
Job failed (attempt 2/5): ... backoff 2000ms
Job failed (attempt 3/5): ... backoff 4000ms
Job failed (attempt 4/5): ... backoff 8000ms
Job completed successfully: ... (attempt 5)
```

## Verify Redis Operations

### Check All Keys
```bash
docker exec $(docker-compose ps -q redis) redis-cli KEYS "*"
```

Output includes:
```
order:{order_id}                 # Cached order
job:notification:{order_id}      # Job status
```

### Monitor Redis in Real-Time
```bash
docker exec $(docker-compose ps -q redis) redis-cli MONITOR
```

### Flush Cache (for testing)
```bash
docker exec $(docker-compose ps -q redis) redis-cli FLUSHDB
```

## Switch Provider Mode

### Test Real SMTP (if you have mail server)

```bash
# Edit .env
PROVIDER_MODE=REAL
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=app-specific-password
SMTP_FROM=noreply@ap2.dev

# Restart
docker-compose restart notification-service
```

### Back to Simulated
```bash
PROVIDER_MODE=SIMULATED
docker-compose restart notification-service
```

## Troubleshooting

### Order Service can't connect to Redis
```bash
# Check Redis is running
docker-compose ps redis

# Check connection
docker exec $(docker-compose ps -q redis) redis-cli PING
# Should output: PONG

# Check REDIS_URL in .env
grep REDIS_URL .env
```

### Notifications not being sent
```bash
# Check RabbitMQ is running
docker-compose ps rabbitmq

# Check notification service is consuming
docker-compose logs notification-service | tail -20

# Check message queue
docker exec $(docker-compose ps -q rabbitmq) rabbitmq-admin list_queues
```

### Cache not working
```bash
# Verify cache is enabled
docker-compose logs order-service | grep -i "cache\|redis"

# Check cache keys exist
docker exec $(docker-compose ps -q redis) redis-cli KEYS "order:*"

# Check cache TTL
docker exec $(docker-compose ps -q redis) redis-cli TTL "order:{order_id}"
```

## Performance Baseline

Run these tests to measure performance improvements:

### Without Cache (First Read)
```bash
time curl http://localhost:8001/orders/$ORDER_ID
# ~50-100ms expected
```

### With Cache (Second Read)
```bash
time curl http://localhost:8001/orders/$ORDER_ID
# ~5-15ms expected (5-10x faster!)
```

### Notification Latency (SIMULATED mode)
```bash
# With SIMULATED_LATENCY_MS=500:
# Notification should take ~500ms + processing

# Without latency (SIMULATED_LATENCY_MS=0):
# Notification should take <50ms + processing
```

## Next Steps

1. **Production Deployment**:
   - Switch PROVIDER_MODE to REAL with valid SMTP credentials
   - Increase CACHE_TTL_SECONDS to 600+ (10 minutes)
   - Set SIMULATED_FAILURE_RATE=0 (disabled)

2. **Monitoring**:
   - Set up Redis monitoring and alerting
   - Track cache hit rates: `docker exec redis redis-cli INFO stats`
   - Monitor job success/failure rates

3. **Scaling**:
   - Redis persistence: Configure RDB/AOF in docker-compose
   - Job processor: Consider multiple consumers for parallel processing
   - Cache distribution: Plan for Redis cluster if needed

---

See `IMPLEMENTATION_GUIDE.md` for comprehensive documentation.
