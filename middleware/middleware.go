package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"turbocache/models"

	"github.com/gin-gonic/gin"
)

func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "Missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "Invalid authorization header format",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(token)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.ErrorResponse{
				Code:    "UNAUTHORIZED",
				Message: "Invalid token",
			})
			return
		}

		c.Next()
	}
}
