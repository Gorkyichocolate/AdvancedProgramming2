package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/app"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("no .env file, reading from environment")
	}

	// Try DATABASE_URL first, then fall back to ORDER_DB_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("ORDER_DB_URL")
	}
	if dbURL == "" {
		log.Fatal("DATABASE_URL or ORDER_DB_URL is required")
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

	paymentGRPCAddr := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentGRPCAddr == "" {
		paymentGRPCAddr = os.Getenv("PAYMENT_GRPC_ADDR")
		if paymentGRPCAddr == "" {
			paymentGRPCAddr = "localhost:9002"
		}
	}

	addr := os.Getenv("ORDER_ADDR")
	if addr == "" {
		addr = ":8001"
	}

	orderGRPCAddr := os.Getenv("ORDER_GRPC_ADDR")
	if orderGRPCAddr == "" {
		orderGRPCAddr = ":9001"
	}

	// Initialize app with cache if Redis URL is provided
	redisURL := os.Getenv("REDIS_URL")
	cacheTTL := 300 // default 5 minutes
	if ttlStr := os.Getenv("CACHE_TTL_SECONDS"); ttlStr != "" {
		if ttl, err := strconv.Atoi(ttlStr); err == nil {
			cacheTTL = ttl
		}
	}

	var application *app.App
	if redisURL != "" {
		log.Println("order-service: initializing with Redis cache")
		var err error
		application, err = app.NewWithCache(db, paymentGRPCAddr, redisURL, cacheTTL)
		if err != nil {
			log.Fatalf("cannot create order app with cache: %v", err)
		}
	} else {
		log.Println("order-service: initializing without cache")
		var err error
		application, err = app.New(db, paymentGRPCAddr)
		if err != nil {
			log.Fatalf("cannot create order app: %v", err)
		}
	}

	defer func() {
		_ = application.Close()
	}()

	if err := application.Run(addr, orderGRPCAddr); err != nil {
		log.Fatalf("order-service exited: %v", err)
	}
}
