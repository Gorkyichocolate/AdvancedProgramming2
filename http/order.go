package http

import "github.com/gin-gonic/gin"

func OrderPost(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Order created successfully",
	})
}
func OrderGet(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Order created successfully",
	})
}
func OrderPatch(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Order created successfully",
	})
}
