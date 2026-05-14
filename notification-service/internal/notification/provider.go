package notification

// NotificationProvider defines the interface for sending notifications
type NotificationProvider interface {
	// SendNotification sends a notification to the recipient
	SendNotification(recipientEmail string, subject string, body string) error
}

// NotificationConfig holds the configuration for notification providers
type NotificationConfig struct {
	// SMTP configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Provider mode
	Mode string

	// Simulated provider configuration
	SimulatedFailureRate  float64
	SimulatedLatencyMs    int
	RetryMaxAttempts      int
	RetryInitialBackoffMs int
	RetryMaxBackoffMs     int
}
