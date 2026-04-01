package app

import (
	"AdvancedProgramming2/order-service/internal/repository"
	"AdvancedProgramming2/order-service/internal/transport"
	"AdvancedProgramming2/order-service/internal/usecase"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	router *gin.Engine
}

func New(db *pgxpool.Pool, paymentServiceURL string) *App {
	httpClient := &stdhttp.Client{Timeout: 2 * time.Second}

	repo := repository.NewOrderPostgresRepo(db)
	paymentClient := transport.NewPaymentHTTPClient(httpClient, paymentServiceURL)
	uc := usecase.NewOrderUsecase(repo, paymentClient)
	handler := transport.NewOrderHandler(uc)

	r := gin.Default()
	orders := r.Group("/orders")
	{
		orders.POST("/", handler.CreateOrder)
		orders.GET("/:id", handler.GetOrder)
		orders.PATCH("/:id/cancel", handler.CancelOrder)
	}

	return &App{router: r}
}

func (a *App) Run(addr string) error {
	return a.router.Run(addr)
}
