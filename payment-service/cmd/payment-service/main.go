package main

import (
	"context"
	"log"
	"os"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/app"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/messaging"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	dbURL := os.Getenv("PAYMENT_DB_URL")
	if dbURL == "" {
		log.Fatal("PAYMENT_DB_URL is required")
	}

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("cannot create db pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("cannot reach payment DB: %v", err)
	}
	log.Println("payment-service: connected to DB")

	httpAddr := getEnv("PAYMENT_ADDR", ":8089")
	grpcAddr := getEnv("PAYMENT_GRPC_ADDR", ":8088")

	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	queue := getEnv("RABBITMQ_QUEUE", "payment.completed")

	publisher, err := messaging.NewRabbitMQPublisher(rabbitURL, queue)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("HTTP running on", httpAddr)
	log.Println("gRPC running on", grpcAddr)

	if err := app.New(db, publisher).Run(httpAddr, grpcAddr); err != nil {
		log.Fatalf("payment-service exited: %v", err)
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
