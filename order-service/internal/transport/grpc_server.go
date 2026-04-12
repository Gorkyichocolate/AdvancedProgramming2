package transport

import (
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/domain"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
)

type OrderGRPCServer struct {
	ap2v1.UnimplementedOrderServiceServer
	uc *usecase.OrderUsecase
}

func NewOrderGRPCServer(uc *usecase.OrderUsecase) *OrderGRPCServer {
	return &OrderGRPCServer{uc: uc}
}

func (s *OrderGRPCServer) SubscribeToOrderUpdates(req *ap2v1.OrderRequest, stream ap2v1.OrderService_SubscribeToOrderUpdatesServer) error {
	if req == nil || req.OrderId == "" {
		return status.Error(codes.InvalidArgument, "order_id is required")
	}

	order, err := s.uc.GetOrder(stream.Context(), req.OrderId)
	if err != nil {
		return status.Error(codes.Internal, "failed to fetch order")
	}
	if order == nil {
		return status.Error(codes.NotFound, "order not found")
	}

	if err := stream.Send(toOrderStatusUpdate(order)); err != nil {
		return err
	}

	updates, cancel := s.uc.SubscribeOrderUpdates(stream.Context(), req.OrderId)
	defer cancel()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case updatedOrder, ok := <-updates:
			if !ok {
				return nil
			}
			if err := stream.Send(toOrderStatusUpdate(updatedOrder)); err != nil {
				return err
			}
		}
	}
}

func toOrderStatusUpdate(order *domain.Order) *ap2v1.OrderStatusUpdate {
	return &ap2v1.OrderStatusUpdate{
		OrderId:    order.ID,
		Status:     order.Status,
		CustomerId: order.CustomerID,
		ItemName:   order.ItemName,
		Amount:     order.Amount,
		UpdatedAt:  timestamppb.Now(),
	}
}
