package transport

import (
	"context"
	"time"

	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentGRPCClient struct {
	conn   *grpc.ClientConn
	client ap2v1.PaymentServiceClient
}

func NewPaymentGRPCClient(ctx context.Context, addr string) (*PaymentGRPCClient, error) {
	// Create a context with timeout for connection establishment
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return &PaymentGRPCClient{
		conn:   conn,
		client: ap2v1.NewPaymentServiceClient(conn),
	}, nil
}

func (p *PaymentGRPCClient) Pay(ctx context.Context, orderID string, amount int64) (*usecase.PaymentResult, error) {
	// Ensure payment call has a timeout (max 10 seconds)
	payCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := p.client.ProcessPayment(payCtx, &ap2v1.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return nil, err
	}

	return &usecase.PaymentResult{
		TransactionID: resp.TransactionId,
		Status:        resp.Status,
	}, nil
}

func (p *PaymentGRPCClient) Close() error {
	return p.conn.Close()
}
