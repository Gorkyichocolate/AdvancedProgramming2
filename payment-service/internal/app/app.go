package app

import (
	"AdvancedProgramming2/payment-service/internal/repository"
	"AdvancedProgramming2/payment-service/internal/transport"
	"AdvancedProgramming2/payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	router *gin.Engine
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

	return &App{router: r}
}

func (a *App) Run(addr string) error {
	return a.router.Run(addr)
}
