package repository

import (
	"context"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentPostgresRepo struct {
	db *pgxpool.Pool
}

func NewPaymentPostgresRepo(db *pgxpool.Pool) *PaymentPostgresRepo {
	return &PaymentPostgresRepo{db: db}
}

func (r *PaymentPostgresRepo) Create(ctx context.Context, p *domain.Payment) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO payments (id, order_id, transaction_id, amount, status)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.ID, p.OrderID, p.TransactionID, p.Amount, p.Status,
	)
	return err
}

func (r *PaymentPostgresRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p := &domain.Payment{}
	err := r.db.QueryRow(ctx,
		`SELECT id, order_id, transaction_id, amount, status
		 FROM payments WHERE order_id = $1`,
		orderID,
	).Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *PaymentPostgresRepo) ListByStatus(ctx context.Context, status string) ([]domain.Payment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, order_id, transaction_id, amount, status
		 FROM payments WHERE status = $1
		 ORDER BY created_at DESC`,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		p := domain.Payment{}
		if err := rows.Scan(&p.ID, &p.OrderID, &p.TransactionID, &p.Amount, &p.Status); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}
