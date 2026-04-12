package app

import (
	"context"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/repository"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/transport"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"
	"net"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type App struct {
	router        *gin.Engine
	grpcServer    *grpc.Server
	paymentClient *transport.PaymentGRPCClient
}

func New(db *pgxpool.Pool, paymentGRPCAddr string) (*App, error) {
	paymentClient, err := transport.NewPaymentGRPCClient(context.Background(), paymentGRPCAddr)
	if err != nil {
		return nil, err
	}

	repo := repository.NewOrderPostgresRepo(db)
	broadcaster := usecase.NewOrderStatusBroadcaster()
	uc := usecase.NewOrderUsecase(repo, paymentClient, broadcaster)
	handler := transport.NewOrderHandler(uc)

	r := gin.Default()
	orders := r.Group("/orders")
	{
		orders.POST("/", handler.CreateOrder)
		orders.GET("/:id", handler.GetOrder)
		orders.PATCH("/:id/cancel", handler.CancelOrder)
		orders.GET("/", handler.GetList)
	}

	grpcServer := grpc.NewServer()
	ap2v1.RegisterOrderServiceServer(grpcServer, transport.NewOrderGRPCServer(uc))

	return &App{router: r, grpcServer: grpcServer, paymentClient: paymentClient}, nil
}

func (a *App) Run(httpAddr, grpcAddr string) error {
	errChan := make(chan error, 2)

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}

	go func() {
		errChan <- a.grpcServer.Serve(listener)
	}()

	go func() {
		errChan <- a.router.Run(httpAddr)
	}()

	return <-errChan
}

func (a *App) Close() error {
	if a.paymentClient == nil {
		return nil
	}
	return a.paymentClient.Close()
}
