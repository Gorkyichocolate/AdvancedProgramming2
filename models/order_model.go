package models

import "time"

type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64  // Amount in cents (e.g., 1000 = $10.00)
	Status     string // "Pending", "Paid", "Failed", "Cancelled"
	CreatedAt  time.Time
}
