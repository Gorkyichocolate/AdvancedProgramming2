package repository

import (
	"context"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderPostgresRepo struct {
	db *pgxpool.Pool
}

func NewOrderPostgresRepo(db *pgxpool.Pool) *OrderPostgresRepo {
	return &OrderPostgresRepo{db: db}
}

func (r *OrderPostgresRepo) Create(ctx context.Context, o *domain.Order) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO orders (id, customer_id, item_name, amount, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		o.ID, o.CustomerID, o.ItemName, o.Amount, o.Status, o.CreatedAt,
	)
	return err
}

func (r *OrderPostgresRepo) OrderList(ctx context.Context, minAmount, maxAmount int64) ([]domain.Order, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at
		 FROM orders
		 WHERE amount >= $1 AND amount <= $2
		 ORDER BY created_at DESC`,
		minAmount, maxAmount,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return orders, nil
}

func (r *OrderPostgresRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	o := &domain.Order{}
	err := r.db.QueryRow(ctx,
		`SELECT id, customer_id, item_name, amount, status, created_at
		 FROM orders WHERE id = $1`,
		id,
	).Scan(&o.ID, &o.CustomerID, &o.ItemName, &o.Amount, &o.Status, &o.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

func (r *OrderPostgresRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE orders SET status = $1 WHERE id = $2`,
		status, id,
	)
	return err
}
