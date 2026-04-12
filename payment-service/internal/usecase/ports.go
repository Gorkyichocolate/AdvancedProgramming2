package usecase

import (
	"context"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/domain"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}
