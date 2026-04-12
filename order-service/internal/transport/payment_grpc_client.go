package transport

import (
	"context"

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
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	return &PaymentGRPCClient{
		conn:   conn,
		client: ap2v1.NewPaymentServiceClient(conn),
	}, nil
}

func (p *PaymentGRPCClient) Pay(ctx context.Context, orderID string, amount int64) (*usecase.PaymentResult, error) {
	resp, err := p.client.ProcessPayment(ctx, &ap2v1.PaymentRequest{
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
