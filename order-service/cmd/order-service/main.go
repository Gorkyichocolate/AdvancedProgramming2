package main

import (
	"AdvancedProgramming2/order-service/internal/app"
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("no .env file, reading from environment")
	}

	dbURL := os.Getenv("ORDER_DB_URL")
	if dbURL == "" {
		log.Fatal("ORDER_DB_URL is required")
	}

	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("cannot create db pool: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("cannot reach order DB: %v", err)
	}
	log.Println("order-service: connected to DB")

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		paymentServiceURL = "http://localhost:8081/payments/"
	}

	addr := os.Getenv("ORDER_ADDR")
	if addr == "" {
		addr = ":8086"
	}

	if err := app.New(db, paymentServiceURL).Run(addr); err != nil {
		log.Fatalf("order-service exited: %v", err)
	}
}
