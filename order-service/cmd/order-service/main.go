package main

import (
	"context"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/app"
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

	paymentGRPCAddr := os.Getenv("PAYMENT_GRPC_ADDR")
	if paymentGRPCAddr == "" {
		paymentGRPCAddr = "localhost:8088"
	}

	addr := os.Getenv("ORDER_ADDR")
	if addr == "" {
		addr = ":8086"
	}

	orderGRPCAddr := os.Getenv("ORDER_GRPC_ADDR")
	if orderGRPCAddr == "" {
		orderGRPCAddr = ":8085"
	}

	app, err := app.New(db, paymentGRPCAddr)
	if err != nil {
		log.Fatalf("cannot create order app: %v", err)
	}
	defer func() {
		_ = app.Close()
	}()

	if err := app.Run(addr, orderGRPCAddr); err != nil {
		log.Fatalf("order-service exited: %v", err)
	}
}
