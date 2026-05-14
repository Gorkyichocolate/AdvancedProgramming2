FROM golang:1.25-alpine AS builder

WORKDIR /app/project

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o payment-service \
    ./payment-service/cmd/payment-service

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/project/payment-service .

CMD ["./payment-service"]
