package notification

type NotificationProvider interface {
	SendNotification(recipientEmail string, subject string, body string) error
}

type NotificationConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	Mode         string

	SimulatedFailureRate  float64
	SimulatedLatencyMs    int
	RetryMaxAttempts      int
	RetryInitialBackoffMs int
	RetryMaxBackoffMs     int
}
