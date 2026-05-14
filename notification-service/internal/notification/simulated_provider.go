package notification

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// SimulatedNotificationProvider simulates sending notifications with configurable latency and failures
type SimulatedNotificationProvider struct {
	failureRate float64 // 0.0 to 1.0
	latencyMs   int     // milliseconds
}

// NewSimulatedNotificationProvider creates a new simulated provider
func NewSimulatedNotificationProvider(failureRate float64, latencyMs int) *SimulatedNotificationProvider {
	if failureRate < 0 {
		failureRate = 0
	}
	if failureRate > 1 {
		failureRate = 1
	}
	if latencyMs < 0 {
		latencyMs = 0
	}

	return &SimulatedNotificationProvider{
		failureRate: failureRate,
		latencyMs:   latencyMs,
	}
}

// SendNotification simulates sending a notification with latency and possible failure
func (sp *SimulatedNotificationProvider) SendNotification(recipientEmail string, subject string, body string) error {
	// Simulate network latency
	if sp.latencyMs > 0 {
		log.Printf("[SIMULATED] Simulating network latency: %dms", sp.latencyMs)
		time.Sleep(time.Duration(sp.latencyMs) * time.Millisecond)
	}

	// Simulate random failure
	if sp.failureRate > 0 {
		randomValue := rand.Float64()
		if randomValue < sp.failureRate {
			err := fmt.Errorf("simulated provider error: random failure (probability: %.2f%%)", sp.failureRate*100)
			log.Printf("[SIMULATED] %v", err)
			return err
		}
	}

	// Log the "sent" notification
	log.Printf("[SIMULATED] Notification sent to: %s", recipientEmail)
	log.Printf("[SIMULATED] Subject: %s", subject)
	log.Printf("[SIMULATED] Body: %s", body)

	return nil
}
