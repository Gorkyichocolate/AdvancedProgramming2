FROM golang:1.25-alpine AS builder

WORKDIR /build

# Копируем go.mod и go.sum
COPY go.mod go.sum ./

# Копируем ap2-generated если он не пустой
COPY ap2-generated/ ./ap2-generated/

# Загружаем зависимости
RUN go mod download

# Копируем весь payment-service
COPY payment-service/ ./payment-service/

# Собираем приложение
WORKDIR /build/payment-service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o payment-service \
    ./cmd/payment-service

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /build/payment-service/payment-service .

EXPOSE 8002 9002

CMD ["./payment-service"]