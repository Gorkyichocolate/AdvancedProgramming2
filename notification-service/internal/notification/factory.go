package notification

import (
	"fmt"
	"strings"
)

// ProviderFactory creates notification providers based on configuration
type ProviderFactory struct {
	config *NotificationConfig
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(config *NotificationConfig) *ProviderFactory {
	return &ProviderFactory{config: config}
}

// CreateProvider creates a notification provider based on the configured mode
func (pf *ProviderFactory) CreateProvider() (NotificationProvider, error) {
	mode := strings.ToLower(strings.TrimSpace(pf.config.Mode))

	switch mode {
	case "simulated", "mock":
		return NewSimulatedNotificationProvider(pf.config.SimulatedFailureRate, pf.config.SimulatedLatencyMs), nil
	case "real", "smtp":
		if pf.config.SMTPHost == "" {
			return nil, fmt.Errorf("SMTP_HOST is required for real provider mode")
		}
		if pf.config.SMTPPort == 0 {
			pf.config.SMTPPort = 587 // Default SMTP port
		}
		if pf.config.SMTPFrom == "" {
			return nil, fmt.Errorf("SMTP_FROM is required for real provider mode")
		}
		return NewSMTPNotificationProvider(
			pf.config.SMTPHost,
			pf.config.SMTPPort,
			pf.config.SMTPUser,
			pf.config.SMTPPassword,
			pf.config.SMTPFrom,
		), nil
	default:
		return nil, fmt.Errorf("unknown provider mode: %s (supported: simulated, real)", mode)
	}
}
