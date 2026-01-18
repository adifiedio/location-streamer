package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// claims extends standard jwt claims with cognito specific fields
type Claims struct {
	jwt.RegisteredClaims
	CognitoGroups []string `json:"cognito:groups"`
	Email         string   `json:"email"`
}

// auth middleware validates the cognito jwt
func Middleware() gin.HandlerFunc {
	jwksURL := os.Getenv("COGNITO_JWKS_URL")
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		logrus.Errorf("Failed to initialize JWKS from URL %s: %v", jwksURL, err)
	}

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
			return
		}

		tokenString := bearerToken[1]

		if jwks == nil {
			// fail safe
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "auth configuration error"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, jwks.Keyfunc)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token signature"})
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			c.Set("user_sub", claims.Subject)
			c.Set("user_email", claims.Email)

			isAdmin := false
			for _, group := range claims.CognitoGroups {
				if group == "Admin" {
					isAdmin = true
					break
				}
			}
			c.Set("is_admin", isAdmin)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		c.Next()
	}
}
