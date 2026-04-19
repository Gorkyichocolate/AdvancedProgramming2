package transport

import (
	"context"

	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/usecase"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentGRPCServer struct {
	ap2v1.UnimplementedPaymentServiceServer
	uc *usecase.PaymentUsecase
}

func NewPaymentGRPCServer(uc *usecase.PaymentUsecase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *ap2v1.PaymentRequest) (*ap2v1.PaymentResponse, error) {
	if req == nil || req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	payment, err := s.uc.ProcessPayment(ctx, req.OrderId, req.Amount)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to process payment")
	}

	return &ap2v1.PaymentResponse{
		TransactionId: payment.TransactionID,
		Status:        payment.Status,
		ProcessedAt:   timestamppb.Now(),
	}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *ap2v1.ListPaymentsRequest) (*ap2v1.ListPaymentsResponse, error) {
	if req == nil || req.Status == "" {
		return nil, status.Error(codes.InvalidArgument, "status is required")
	}

	payments, err := s.uc.ListPaymentsByStatus(ctx, req.Status)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list payments")
	}

	var pbPayments []*ap2v1.PaymentResponse
	for _, p := range payments {
		pbPayments = append(pbPayments, &ap2v1.PaymentResponse{
			TransactionId: p.TransactionID,
			Status:        p.Status,
			ProcessedAt:   timestamppb.Now(),
		})
	}

	return &ap2v1.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
