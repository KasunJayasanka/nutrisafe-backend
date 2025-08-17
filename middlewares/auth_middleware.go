// middlewares/auth_middleware.go
package middlewares

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"backend/config"
	"backend/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		secret := []byte(os.Getenv("JWT_SECRET"))
		if len(secret) == 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfigured: JWT_SECRET not set"})
			return
		}

		// Parse & validate HS256 by default (adjust if you use RS256, etc.)
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}

		// 1) Prefer userId claim if your token includes it
		if v, ok := claims["userId"]; ok {
			switch id := v.(type) {
			case float64: // common when JWT was JSON-encoded
				c.Set("userID", uint(id))
				c.Next()
				return
			case int64:
				c.Set("userID", uint(id))
				c.Next()
				return
			case string:
				// if you encode as string, convert to uint as needed
				// ... parse here if you want ...
			}
		}

		// 2) Fallback: use email claim and look up DB
		email, _ := claims["email"].(string)
		if email == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "email claim missing"})
			return
		}

		var user models.User
		if err := config.DB.Where("email = ?", email).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// Set both for convenience
		c.Set("userID", uint(user.ID))
		c.Set("email", email)

		c.Next()
	}
}
