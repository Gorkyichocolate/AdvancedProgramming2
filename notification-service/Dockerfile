FROM golang:1.25-alpine AS builder

WORKDIR /app/project

COPY . .

RUN go build -o notification ./notification-service/cmd/notification-service

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -o notification \
    ./notification-service/cmd/notification-service

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/project/notification .

CMD ["./notification"]