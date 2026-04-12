package app

import (
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/repository"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/transport"
	"github.com/Gorkyichocolate/AdvancedProgramming2/payment-service/internal/usecase"
	"net"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

type App struct {
	router     *gin.Engine
	grpcServer *grpc.Server
}

func New(db *pgxpool.Pool) *App {
	repo := repository.NewPaymentPostgresRepo(db)
	uc := usecase.NewPaymentUsecase(repo)
	handler := transport.NewPaymentHandler(uc)

	r := gin.Default()
	payments := r.Group("/payments")
	{
		payments.POST("/", handler.ProcessPayment)
		payments.GET("/:id", handler.GetPayment)
	}

	grpcServer := grpc.NewServer()
	ap2v1.RegisterPaymentServiceServer(grpcServer, transport.NewPaymentGRPCServer(uc))

	return &App{router: r, grpcServer: grpcServer}
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
