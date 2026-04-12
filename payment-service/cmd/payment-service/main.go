package main

import (
	"context"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/app"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("no .env file, reading from environment")
	}

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

	addr := os.Getenv("PAYMENT_ADDR")
	if addr == "" {
		addr = ":8087"
	}

	grpcAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = ":8088"
	}

	if err := app.New(db).Run(addr, grpcAddr); err != nil {
		log.Fatalf("payment-service exited: %v", err)
	}
}
