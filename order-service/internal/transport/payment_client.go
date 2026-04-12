package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"
	"net/http"
)

type PaymentHTTPClient struct {
	client  *http.Client
	baseURL string
}

func NewPaymentHTTPClient(client *http.Client, baseURL string) *PaymentHTTPClient {
	return &PaymentHTTPClient{client: client, baseURL: baseURL}
}

type paymentResp struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

func (p *PaymentHTTPClient) Pay(ctx context.Context, orderID string, amount int64) (*usecase.PaymentResult, error) {
	body, _ := json.Marshal(map[string]any{
		"order_id": orderID,
		"amount":   amount,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		// Network error or timeout — the http.Client timeout (2s) trips here.
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New("payment service returned non-201 status")
	}

	var pr paymentResp
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return &usecase.PaymentResult{
		TransactionID: pr.TransactionID,
		Status:        pr.Status,
	}, nil
}
