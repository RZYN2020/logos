// Package middleware 中间件测试
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupTestAuthConfig() *AuthConfig {
	return &AuthConfig{
		SecretKey: "test-secret-key",
	}
}

// TestGenerateToken 测试生成 token
func TestGenerateToken(t *testing.T) {
	auth := setupTestAuthConfig()

	token, err := auth.GenerateToken("user-001", "testuser", []string{"user"})

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestParseToken 测试解析 token
func TestParseToken(t *testing.T) {
	auth := setupTestAuthConfig()

	// 生成 token
	tokenString, _ := auth.GenerateToken("user-001", "testuser", []string{"user"})

	// 解析 token
	claims, err := auth.ParseToken(tokenString)

	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-001", claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Contains(t, claims.Roles, "user")
}

// TestParseInvalidToken 测试解析无效 token
func TestParseInvalidToken(t *testing.T) {
	auth := setupTestAuthConfig()

	claims, err := auth.ParseToken("invalid-token")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

// TestAuthMiddleware 测试认证中间件
func TestAuthMiddleware(t *testing.T) {
	auth := setupTestAuthConfig()
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	router.Use(auth.AuthMiddleware())
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试没有 authorization header
	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 测试无效的 authorization 格式
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Invalid format")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)

	// 测试无效的 token
	req3, _ := http.NewRequest("GET", "/protected", nil)
	req3.Header.Set("Authorization", "Bearer invalid-token")
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusUnauthorized, w3.Code)

	// 测试有效的 token
	tokenString, _ := auth.GenerateToken("user-001", "testuser", []string{"user"})
	req4, _ := http.NewRequest("GET", "/protected", nil)
	req4.Header.Set("Authorization", "Bearer "+tokenString)
	w4 := httptest.NewRecorder()
	router.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}

// TestRequireRole 测试角色验证中间件
func TestRequireRole(t *testing.T) {
	auth := setupTestAuthConfig()
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	router.Use(auth.AuthMiddleware())
	router.GET("/admin", auth.RequireRole("admin"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access"})
	})

	// 测试普通用户访问管理员接口
	tokenString, _ := auth.GenerateToken("user-001", "testuser", []string{"user"})
	req, _ := http.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)

	// 测试管理员访问
	adminToken, _ := auth.GenerateToken("admin-001", "admin", []string{"admin"})
	req2, _ := http.NewRequest("GET", "/admin", nil)
	req2.Header.Set("Authorization", "Bearer "+adminToken)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestExpiredToken 测试过期 token
func TestExpiredToken(t *testing.T) {
	auth := setupTestAuthConfig()

	// 创建已过期的 token
	claims := Claims{
		UserID:   "user-001",
		Username: "testuser",
		Roles:    []string{"user"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "logos-log-analyzer",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(auth.SecretKey))

	// 解析过期 token
	parsedClaims, err := auth.ParseToken(tokenString)

	assert.Error(t, err)
	assert.Nil(t, parsedClaims)
}

// TestCorsMiddleware 测试 CORS 中间件
func TestCorsMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.Default()
	router.Use(CorsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 测试 OPTIONS 请求
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// 测试 GET 请求
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "*", w2.Header().Get("Access-Control-Allow-Origin"))
}
