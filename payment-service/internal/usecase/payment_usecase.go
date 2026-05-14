package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/domain"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/events"
)

var ErrNotFound = errors.New("payment not found")

type PaymentUsecase struct {
	repo      PaymentRepository
	publisher EventPublisher
}

func NewPaymentUsecase(repo PaymentRepository, publisher EventPublisher) *PaymentUsecase {
	return &PaymentUsecase{
		repo:      repo,
		publisher: publisher,
	}
}

func (u *PaymentUsecase) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	status := domain.StatusAuthorized
	if amount > domain.PaymentLimit {
		status = domain.StatusDeclined
	}

	p := &domain.Payment{
		ID:            newUUID(),
		OrderID:       orderID,
		TransactionID: newUUID(),
		Amount:        amount,
		Status:        status,
	}

	if err := u.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	if p.Status == domain.StatusAuthorized {
		event := events.PaymentEvent{
			OrderID:       p.OrderID,
			Amount:        float64(p.Amount),
			CustomerEmail: "test@example.com",
			Status:        "completed",
			MessageID:     newUUID(),
		}

		if err := u.publisher.PublishPaymentCompleted(ctx, event); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (u *PaymentUsecase) GetPayment(ctx context.Context, orderID string) (*domain.Payment, error) {
	p, err := u.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrNotFound
	}
	return p, nil
}

func (u *PaymentUsecase) ListPaymentsByStatus(ctx context.Context, status string) ([]domain.Payment, error) {
	payments, err := u.repo.ListByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	return payments, nil
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
