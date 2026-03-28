package http

import "github.com/gin-gonic/gin"

func PaymentPost(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Payment created successfully",
	})
}
func PaymentGet(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Payment created successfully",
	})
}
