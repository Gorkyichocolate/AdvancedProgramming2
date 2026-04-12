package usecase

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/domain"
	"time"
)

var (
	ErrInvalidAmount             = errors.New("amount must be greater than zero")
	ErrInvalidAmountRange        = errors.New("min_amount and max_amount must be in range [1000, 50000] and min_amount <= max_amount")
	ErrNotFound                  = errors.New("order not found")
	ErrNotCancellable            = errors.New("only Pending orders can be cancelled")
	ErrPaymentServiceUnavailable = errors.New("payment service unavailable")
)

type OrderUsecase struct {
	repo          OrderRepository
	paymentClient PaymentClient
	broadcaster   *OrderStatusBroadcaster
}

func NewOrderUsecase(repo OrderRepository, paymentClient PaymentClient, broadcaster *OrderStatusBroadcaster) *OrderUsecase {
	return &OrderUsecase{repo: repo, paymentClient: paymentClient, broadcaster: broadcaster}
}

func (u *OrderUsecase) CreateOrder(ctx context.Context, customerID, itemName string, amount int64) (*domain.Order, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	order := &domain.Order{
		ID:         newUUID(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := u.repo.Create(ctx, order); err != nil {
		return nil, err
	}
	u.publishOrder(order)

	result, err := u.paymentClient.Pay(ctx, order.ID, order.Amount)
	if err != nil {
		if updateErr := u.repo.UpdateStatus(ctx, order.ID, domain.StatusFailed); updateErr != nil {
			return nil, updateErr
		}
		order.Status = domain.StatusFailed
		u.publishOrder(order)
		return order, ErrPaymentServiceUnavailable
	}

	newStatus := domain.StatusFailed
	if result.Status == "Authorized" {
		newStatus = domain.StatusPaid
	}

	if err := u.repo.UpdateStatus(ctx, order.ID, newStatus); err != nil {
		return nil, err
	}
	order.Status = newStatus
	u.publishOrder(order)
	return order, nil
}

func (u *OrderUsecase) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrNotFound
	}
	return order, nil
}

func (u *OrderUsecase) CancelOrder(ctx context.Context, id string) (*domain.Order, error) {
	order, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrNotFound
	}
	if order.Status != domain.StatusPending {
		return nil, ErrNotCancellable
	}
	if err := u.repo.UpdateStatus(ctx, id, domain.StatusCancelled); err != nil {
		return nil, err
	}
	order.Status = domain.StatusCancelled
	u.publishOrder(order)
	return order, nil
}

func (u *OrderUsecase) SubscribeOrderUpdates(ctx context.Context, orderID string) (<-chan *domain.Order, func()) {
	updates, cancel := u.broadcaster.Subscribe(ctx)
	filtered := make(chan *domain.Order, 16)

	go func() {
		defer close(filtered)
		for {
			select {
			case <-ctx.Done():
				return
			case order, ok := <-updates:
				if !ok {
					return
				}
				if order.ID == orderID {
					filtered <- order
				}
			}
		}
	}()

	return filtered, cancel
}

func (u *OrderUsecase) publishOrder(order *domain.Order) {
	if u.broadcaster != nil {
		u.broadcaster.Publish(order)
	}
}

func (u *OrderUsecase) OrderList(ctx context.Context, minAmount, maxAmount int64) ([]domain.Order, error) {
	if minAmount < 1 || maxAmount > 100000000 || minAmount > maxAmount {
		return nil, ErrInvalidAmountRange
	}

	orders, err := u.repo.OrderList(ctx, minAmount, maxAmount)
	if err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrNotFound
	}

	return orders, nil
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
