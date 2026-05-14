package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// JobProcessor handles background notification jobs with retry logic and idempotency
type JobProcessor struct {
	provider         NotificationProvider
	redisClient      *redis.Client
	maxRetries       int
	initialBackoffMs int
	maxBackoffMs     int
}

// NotificationJob represents a notification job to be processed
type NotificationJob struct {
	PaymentID      string `json:"payment_id"`
	RecipientEmail string `json:"recipient_email"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
}

// JobStatus represents the status of a job
type JobStatus struct {
	Status       string    `json:"status"` // pending, success, failed
	LastAttempt  time.Time `json:"last_attempt"`
	AttemptCount int       `json:"attempt_count"`
	NextRetry    time.Time `json:"next_retry,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(provider NotificationProvider, redisURL string, maxRetries int, initialBackoffMs int, maxBackoffMs int) (*JobProcessor, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &JobProcessor{
		provider:         provider,
		redisClient:      client,
		maxRetries:       maxRetries,
		initialBackoffMs: initialBackoffMs,
		maxBackoffMs:     maxBackoffMs,
	}, nil
}

// getJobStatusKey generates a Redis key for a job status
func (jp *JobProcessor) getJobStatusKey(paymentID string) string {
	return fmt.Sprintf("job:notification:%s", paymentID)
}

// ProcessJob processes a notification job with idempotency and retry logic
func (jp *JobProcessor) ProcessJob(ctx context.Context, job *NotificationJob) error {
	statusKey := jp.getJobStatusKey(job.PaymentID)

	// Check if job has been processed before
	statusJSON, err := jp.redisClient.Get(ctx, statusKey).Result()
	if err == nil {
		var status JobStatus
		if err := json.Unmarshal([]byte(statusJSON), &status); err == nil {
			// Job already processed
			if status.Status == "success" {
				log.Printf("Job already processed successfully: %s", job.PaymentID)
				return nil
			}
			// Check if we can retry
			if status.AttemptCount >= jp.maxRetries {
				log.Printf("Max retries exceeded for job: %s", job.PaymentID)
				status.Status = "failed"
				jp.updateJobStatus(ctx, statusKey, &status)
				return fmt.Errorf("max retries exceeded")
			}
			// Check if we should retry now
			if time.Now().Before(status.NextRetry) {
				log.Printf("Job not ready for retry yet: %s, retrying at %s", job.PaymentID, status.NextRetry)
				return fmt.Errorf("job not ready for retry")
			}
		}
	} else if err != redis.Nil {
		log.Printf("Error checking job status: %v", err)
	}

	// Get or initialize job status
	var status JobStatus
	if err == nil && statusJSON != "" {
		json.Unmarshal([]byte(statusJSON), &status)
	} else {
		status = JobStatus{Status: "pending", AttemptCount: 0}
	}

	// Attempt to send notification
	err = jp.provider.SendNotification(job.RecipientEmail, job.Subject, job.Body)
	status.LastAttempt = time.Now()
	status.AttemptCount++

	if err != nil {
		// Calculate next retry time with exponential backoff
		backoffMs := int(math.Min(
			float64(jp.initialBackoffMs*int(math.Pow(2, float64(status.AttemptCount-1)))),
			float64(jp.maxBackoffMs),
		))
		status.NextRetry = time.Now().Add(time.Duration(backoffMs) * time.Millisecond)
		status.Error = err.Error()
		status.Status = "pending"

		log.Printf("Job failed (attempt %d/%d): %s. Next retry at %s with backoff %dms",
			status.AttemptCount, jp.maxRetries, job.PaymentID, status.NextRetry, backoffMs)

		jp.updateJobStatus(ctx, statusKey, &status)
		return fmt.Errorf("notification send failed: %w", err)
	}

	// Mark as successful
	status.Status = "success"
	status.Error = ""
	log.Printf("Job completed successfully: %s (attempt %d)", job.PaymentID, status.AttemptCount)

	// Update status - keep for 24 hours to prevent reprocessing
	jp.updateJobStatusWithTTL(ctx, statusKey, &status, 24*time.Hour)
	return nil
}

// updateJobStatus updates the job status in Redis
func (jp *JobProcessor) updateJobStatus(ctx context.Context, key string, status *JobStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	// Set with 7-day TTL to prevent accumulation of old jobs
	if err := jp.redisClient.Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// updateJobStatusWithTTL updates the job status in Redis with custom TTL
func (jp *JobProcessor) updateJobStatusWithTTL(ctx context.Context, key string, status *JobStatus, ttl time.Duration) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := jp.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// GetJobStatus retrieves the status of a job
func (jp *JobProcessor) GetJobStatus(ctx context.Context, paymentID string) (*JobStatus, error) {
	statusKey := jp.getJobStatusKey(paymentID)
	statusJSON, err := jp.redisClient.Get(ctx, statusKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Job not found
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var status JobStatus
	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	return &status, nil
}

// Close closes the Redis connection
func (jp *JobProcessor) Close() error {
	return jp.redisClient.Close()
}
