package app

import (
	"context"
	"log"
	"net"

	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/cache"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/repository"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/transport"
	"github.com/Gorkyichocolate/AdvancedProgramming2/order-service/internal/usecase"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type App struct {
	router        *gin.Engine
	grpcServer    *grpc.Server
	paymentClient *transport.PaymentGRPCClient
	cache         *cache.OrderCache
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

// NewWithCache creates an App with Redis cache enabled
func NewWithCache(db *pgxpool.Pool, paymentGRPCAddr string, redisURL string, cacheTTL int) (*App, error) {
	paymentClient, err := transport.NewPaymentGRPCClient(context.Background(), paymentGRPCAddr)
	if err != nil {
		return nil, err
	}

	orderCache, err := cache.NewOrderCache(redisURL, cacheTTL)
	if err != nil {
		log.Printf("Warning: failed to initialize Redis cache: %v. Continuing without cache.", err)
		// Continue without cache if Redis fails to initialize
		return New(db, paymentGRPCAddr)
	}

	repo := repository.NewOrderPostgresRepo(db)
	broadcaster := usecase.NewOrderStatusBroadcaster()
	uc := usecase.NewOrderUsecase(repo, paymentClient, broadcaster)
	handler := transport.NewOrderHandlerWithCache(uc, orderCache)

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

	return &App{router: r, grpcServer: grpcServer, paymentClient: paymentClient, cache: orderCache}, nil
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
	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			log.Printf("Error closing cache: %v", err)
		}
	}
	if a.paymentClient == nil {
		return nil
	}
	return a.paymentClient.Close()
}
