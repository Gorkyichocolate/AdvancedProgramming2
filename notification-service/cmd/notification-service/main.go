package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/Gorkyichocolate/AdvancedProgramming2/notification-service/internal/notification"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentEvent struct {
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	CustomerEmail string  `json:"customer_email"`
	Status        string  `json:"status"`
	MessageID     string  `json:"message_id"`
}

func main() {
	_ = godotenv.Load(".env")

	// Load configuration
	rabbitURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	queueName := getEnv("RABBITMQ_QUEUE", "payment.completed")
	dlqName := getEnv("RABBITMQ_DLQ", "payment.dlq")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

	// Provider configuration
	providerMode := getEnv("PROVIDER_MODE", "SIMULATED")
	smtpHost := getEnv("SMTP_HOST", "localhost")
	smtpPort := getEnvInt("SMTP_PORT", 1025)
	smtpUser := getEnv("SMTP_USER", "")
	smtpPassword := getEnv("SMTP_PASSWORD", "")
	smtpFrom := getEnv("SMTP_FROM", "noreply@ap2.dev")
	simulatedFailureRate := getEnvFloat64("SIMULATED_FAILURE_RATE", 0.2)
	simulatedLatencyMs := getEnvInt("SIMULATED_LATENCY_MS", 500)

	// Retry configuration
	maxRetries := getEnvInt("PROVIDER_RETRY_MAX_ATTEMPTS", 5)
	initialBackoffMs := getEnvInt("PROVIDER_RETRY_INITIAL_BACKOFF_MS", 2000)
	maxBackoffMs := getEnvInt("PROVIDER_RETRY_MAX_BACKOFF_MS", 32000)

	// Create notification provider based on mode
	config := &notification.NotificationConfig{
		Mode:                  providerMode,
		SMTPHost:              smtpHost,
		SMTPPort:              smtpPort,
		SMTPUser:              smtpUser,
		SMTPPassword:          smtpPassword,
		SMTPFrom:              smtpFrom,
		SimulatedFailureRate:  simulatedFailureRate,
		SimulatedLatencyMs:    simulatedLatencyMs,
		RetryMaxAttempts:      maxRetries,
		RetryInitialBackoffMs: initialBackoffMs,
		RetryMaxBackoffMs:     maxBackoffMs,
	}

	factory := notification.NewProviderFactory(config)
	provider, err := factory.CreateProvider()
	if err != nil {
		log.Fatalf("Failed to create notification provider: %v", err)
	}
	log.Printf("Notification provider initialized: mode=%s", providerMode)

	// Create job processor with Redis backend
	jobProcessor, err := notification.NewJobProcessor(provider, redisURL, maxRetries, initialBackoffMs, maxBackoffMs)
	if err != nil {
		log.Fatalf("Failed to create job processor: %v", err)
	}
	defer jobProcessor.Close()
	log.Println("Job processor initialized with Redis backend")

	// Connect to RabbitMQ
	conn, err := amqp.Dial(rabbitURL)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	_, err = ch.QueueDeclare(
		dlqName,
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare DLQ")

	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		},
	)
	failOnError(err, "Failed to declare queue")

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register consumer")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Notification Service started, waiting for messages...")

	go func() {
		for msg := range msgs {
			var event PaymentEvent

			err := json.Unmarshal(msg.Body, &event)
			if err != nil {
				log.Printf("Invalid message: %v", err)
				msg.Nack(false, false) // Don't requeue invalid messages
				continue
			}

			// Create notification job
			job := &notification.NotificationJob{
				PaymentID:      event.OrderID,
				RecipientEmail: event.CustomerEmail,
				Subject:        "Payment Confirmation",
				Body:           buildEmailBody(event),
			}

			// Process the job (with retry logic and idempotency)
			ctx := context.Background()
			err = jobProcessor.ProcessJob(ctx, job)
			if err != nil {
				log.Printf("Job processing failed for payment %s: %v. Will retry later.", event.OrderID, err)
				// Nack and requeue for later retry
				msg.Nack(false, true)
				continue
			}

			// Acknowledge successful processing
			msg.Ack(false)
			log.Printf("Message processed successfully: payment_id=%s", event.OrderID)
		}
	}()

	<-sigs
	log.Println("Shutting down Notification Service...")
}

func buildEmailBody(event PaymentEvent) string {
	return "Payment confirmation email\n" +
		"Order ID: " + event.OrderID + "\n" +
		"Amount: $" + strconv.FormatFloat(event.Amount, 'f', 2, 64) + "\n" +
		"Status: " + event.Status
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	if intVal, err := strconv.Atoi(val); err == nil {
		return intVal
	}
	return fallback
}

func getEnvFloat64(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal
	}
	return fallback
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
