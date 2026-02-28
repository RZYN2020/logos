// Package middleware HTTP 中间件
package middleware

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 声明
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// AuthConfig 认证配置
type AuthConfig struct {
	SecretKey string
}

// NewAuthConfig 创建认证配置
func NewAuthConfig() *AuthConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "logos-default-secret-key-change-in-production"
	}
	return &AuthConfig{
		SecretKey: secret,
	}
}

// GenerateToken 生成 JWT token
func (c *AuthConfig) GenerateToken(userID, username string, roles []string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "logos-log-analyzer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.SecretKey))
}

// ParseToken 解析 JWT token
func (c *AuthConfig) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, nil
}

// AuthMiddleware JWT 认证中间件
func (c *AuthConfig) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			ctx.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			ctx.Abort()
			return
		}

		claims, err := c.ParseToken(parts[1])
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			ctx.Abort()
			return
		}

		// 设置用户信息到上下文
		ctx.Set("user_id", claims.UserID)
		ctx.Set("username", claims.Username)
		ctx.Set("roles", claims.Roles)

		ctx.Next()
	}
}

// RequireRole 需要特定角色的中间件
func (c *AuthConfig) RequireRole(requiredRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		roles, exists := ctx.Get("roles")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
			ctx.Abort()
			return
		}

		userRoles := roles.([]string)
		hasRole := false
		for _, role := range userRoles {
			if role == requiredRole || role == "admin" {
				hasRole = true
				break
			}
		}

		if !hasRole {
			ctx.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}

// CorsMiddleware CORS 中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User")
		ctx.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		ctx.Writer.Header().Set("Access-Control-Max-Age", "86400")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
