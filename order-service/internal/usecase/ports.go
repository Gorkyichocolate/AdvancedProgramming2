package usecase

import (
	"AdvancedProgramming2/order-service/internal/domain"
	"context"
)

type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, id, status string) error
	OrderList(ctx context.Context, minAmount, maxAmount int64) ([]domain.Order, error)
}

type PaymentResult struct {
	TransactionID string
	Status        string // "Authorized" | "Declined"
}

type PaymentClient interface {
	Pay(ctx context.Context, orderID string, amount int64) (*PaymentResult, error)
}
