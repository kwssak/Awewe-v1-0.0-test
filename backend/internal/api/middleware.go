package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bad token"})
			return
		}
		c.Next()
	}
}
