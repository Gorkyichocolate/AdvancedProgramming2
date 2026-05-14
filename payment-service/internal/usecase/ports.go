package usecase

import (
	"context"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/domain"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/events"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
	ListByStatus(ctx context.Context, status string) ([]domain.Payment, error)
}
type EventPublisher interface {
	PublishPaymentCompleted(ctx context.Context, event events.PaymentEvent) error
}
