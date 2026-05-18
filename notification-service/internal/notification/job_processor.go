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

type JobProcessor struct {
	provider         NotificationProvider
	redisClient      *redis.Client
	maxRetries       int
	initialBackoffMs int
	maxBackoffMs     int
}

type NotificationJob struct {
	PaymentID      string `json:"payment_id"`
	RecipientEmail string `json:"recipient_email"`
	Subject        string `json:"subject"`
	Body           string `json:"body"`
}

type JobStatus struct {
	Status       string    `json:"status"`
	LastAttempt  time.Time `json:"last_attempt"`
	AttemptCount int       `json:"attempt_count"`
	NextRetry    time.Time `json:"next_retry,omitempty"`
	Error        string    `json:"error,omitempty"`
}

func NewJobProcessor(provider NotificationProvider, redisURL string, maxRetries int, initialBackoffMs int, maxBackoffMs int) (*JobProcessor, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opt)

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

func (jp *JobProcessor) getJobStatusKey(paymentID string) string {
	return fmt.Sprintf("job:notification:%s", paymentID)
}

func (jp *JobProcessor) ProcessJob(ctx context.Context, job *NotificationJob) error {
	statusKey := jp.getJobStatusKey(job.PaymentID)

	statusJSON, err := jp.redisClient.Get(ctx, statusKey).Result()
	if err == nil {
		var status JobStatus
		if err := json.Unmarshal([]byte(statusJSON), &status); err == nil {

			if status.Status == "success" {
				log.Printf("[PROCESSOR] [%s] Job already processed successfully: %s",
					time.Now().Format("15:04:05.000"), job.PaymentID)
				return nil
			}

			if status.AttemptCount >= jp.maxRetries {
				log.Printf("[PROCESSOR] [%s] Max retries exceeded for job: %s",
					time.Now().Format("15:04:05.000"), job.PaymentID)
				status.Status = "failed"
				jp.updateJobStatus(ctx, statusKey, &status)
				return fmt.Errorf("max retries exceeded")
			}

			if time.Now().Before(status.NextRetry) {
				log.Printf("[PROCESSOR] [%s] Job not ready for retry yet: %s, retrying at %s",
					time.Now().Format("15:04:05.000"), job.PaymentID, status.NextRetry.Format("15:04:05.000"))
				return fmt.Errorf("job not ready for retry")
			}
		}
	} else if err != redis.Nil {
		log.Printf("[PROCESSOR] [%s] Error checking job status: %v",
			time.Now().Format("15:04:05.000"), err)
	}

	var status JobStatus
	if err == nil && statusJSON != "" {
		json.Unmarshal([]byte(statusJSON), &status)
	} else {
		status = JobStatus{Status: "pending", AttemptCount: 0}
	}

	log.Printf("[PROCESSOR] [%s] Starting notification attempt for payment: %s (attempt %d/%d)",
		time.Now().Format("15:04:05.000"), job.PaymentID, status.AttemptCount+1, jp.maxRetries)

	err = jp.provider.SendNotification(job.RecipientEmail, job.Subject, job.Body)
	status.LastAttempt = time.Now()
	status.AttemptCount++

	if err != nil {

		backoffMs := int(math.Min(
			float64(jp.initialBackoffMs*int(math.Pow(2, float64(status.AttemptCount-1)))),
			float64(jp.maxBackoffMs),
		))
		status.NextRetry = time.Now().Add(time.Duration(backoffMs) * time.Millisecond)
		status.Error = err.Error()
		status.Status = "pending"

		log.Printf("[PROCESSOR] [%s] Job failed (attempt %d/%d) for payment: %s. Next retry at %s with backoff %dms",
			time.Now().Format("15:04:05.000"), status.AttemptCount, jp.maxRetries, job.PaymentID, status.NextRetry.Format("15:04:05.000"), backoffMs)

		jp.updateJobStatus(ctx, statusKey, &status)
		return fmt.Errorf("notification send failed: %w", err)
	}

	status.Status = "success"
	status.Error = ""
	log.Printf("[PROCESSOR] [%s] ✓ Job completed successfully for payment: %s (attempt %d)",
		time.Now().Format("15:04:05.000"), job.PaymentID, status.AttemptCount)

	jp.updateJobStatusWithTTL(ctx, statusKey, &status, 24*time.Hour)
	return nil
}

func (jp *JobProcessor) updateJobStatus(ctx context.Context, key string, status *JobStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	if err := jp.redisClient.Set(ctx, key, data, 7*24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

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

func (jp *JobProcessor) Close() error {
	return jp.redisClient.Close()
}
