package notification

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type SimulatedNotificationProvider struct {
	failureRate float64
	latencyMs   int
}

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

func (sp *SimulatedNotificationProvider) SendNotification(recipientEmail string, subject string, body string) error {
	startTime := time.Now()

	if sp.latencyMs > 0 {
		log.Printf("[SIMULATED] [%s] Simulating network latency: %dms",
			time.Now().Format("15:04:05.000"), sp.latencyMs)
		time.Sleep(time.Duration(sp.latencyMs) * time.Millisecond)
	}

	if sp.failureRate > 0 {
		randomValue := rand.Float64()
		if randomValue < sp.failureRate {
			duration := time.Since(startTime)
			err := fmt.Errorf("simulated provider error: random failure (probability: %.0f%%) after %v",
				sp.failureRate*100, duration)
			log.Printf("[SIMULATED] [%s] ✗ %v to %s",
				time.Now().Format("15:04:05.000"), err, recipientEmail)
			return err
		}
	}

	duration := time.Since(startTime)
	log.Printf("[SIMULATED] [%s] ✓ Notification sent to: %s (in %v)",
		time.Now().Format("15:04:05.000"), recipientEmail, duration)

	return nil
}
