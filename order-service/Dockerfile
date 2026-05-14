FROM golang:1.25-alpine AS builder

WORKDIR /app/project

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o order-service \
    ./order-service/cmd/order-service

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/project/order-service .

CMD ["./order-service"]
