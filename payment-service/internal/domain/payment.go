package domain

const (
	StatusAuthorized       = "Authorized"
	StatusDeclined         = "Declined"
	PaymentLimit     int64 = 100000
)

type Payment struct {
	ID            string `json:"id"`
	OrderID       string `json:"order_id"`
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"` // cents; never float64
	Status        string `json:"status"`
}
