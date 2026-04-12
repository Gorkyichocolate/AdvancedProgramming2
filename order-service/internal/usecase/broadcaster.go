package usecase

import (
	"context"
	"sync"

	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/domain"
)

type OrderStatusBroadcaster struct {
	mu          sync.RWMutex
	nextID      int
	subscribers map[int]chan *domain.Order
}

func NewOrderStatusBroadcaster() *OrderStatusBroadcaster {
	return &OrderStatusBroadcaster{
		subscribers: make(map[int]chan *domain.Order),
	}
}

func (b *OrderStatusBroadcaster) Subscribe(ctx context.Context) (<-chan *domain.Order, func()) {
	ch := make(chan *domain.Order, 16)
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subscribers[id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		if _, ok := b.subscribers[id]; ok {
			delete(b.subscribers, id)
			close(ch)
		}
		b.mu.Unlock()
	}

	go func() {
		<-ctx.Done()
		cancel()
	}()

	return ch, cancel
}

func (b *OrderStatusBroadcaster) Publish(order *domain.Order) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- order:
		default:
		}
	}
}
