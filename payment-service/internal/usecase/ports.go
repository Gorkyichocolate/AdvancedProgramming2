package usecase

import (
	"AdvancedProgramming2/payment-service/internal/domain"
	"context"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}
