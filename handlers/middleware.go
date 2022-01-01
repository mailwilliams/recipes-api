package handlers

import (
	"github.com/gin-gonic/gin"
	"os"
)

func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-API-KEY") != os.Getenv("X_API_KEY") {
			c.AbortWithStatus(401)
		}
		c.Next()
	}
}
