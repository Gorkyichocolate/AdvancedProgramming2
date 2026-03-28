package main

import (
	"AdvancedProgramming2/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	//order
	order := r.Group("/order")

	{
		order.POST("/", http.OrderPost)
		order.GET("/:id", http.OrderGet)
		order.PATCH("/:id/cancel", http.OrderPatch)
	}

	payment := r.Group("/payments")
	{
		payment.POST("/", http.PaymentPost)
		payment.GET("/:id", http.PaymentGet)
	}

	r.Run(":8081")
}
