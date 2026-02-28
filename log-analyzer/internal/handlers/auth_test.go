// Package handlers 认证处理器测试
package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/log-system/log-analyzer/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func setupAuthHandler(t *testing.T) (*AuthHandler, *gin.Engine) {
	authConfig := middleware.NewAuthConfig()
	handler := NewAuthHandler(authConfig)

	router := gin.Default()
	router.POST("/api/v1/auth/login", handler.Login)
	router.POST("/api/v1/auth/register", handler.Register)
	router.GET("/api/v1/user", handler.GetCurrentUser)

	return handler, router
}

// TestLogin 测试用户登录
func TestLogin(t *testing.T) {
	_, router := setupAuthHandler(t)

	loginJSON := `{
		"username": "admin",
		"password": "admin123"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["token"])
	assert.NotNil(t, response["user"])
}

// TestLoginInvalidCredentials 测试登录失败（无效凭证）
func TestLoginInvalidCredentials(t *testing.T) {
	_, router := setupAuthHandler(t)

	loginJSON := `{
		"username": "admin",
		"password": "wrongpassword"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestRegister 测试用户注册
func TestRegister(t *testing.T) {
	_, router := setupAuthHandler(t)

	registerJSON := `{
		"username": "newuser",
		"password": "password123",
		"email": "newuser@example.com"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotNil(t, response["token"])
	assert.NotNil(t, response["user"])
}

// TestRegisterDuplicateUsername 测试注册用户已存在
func TestRegisterDuplicateUsername(t *testing.T) {
	_, router := setupAuthHandler(t)

	// 第一次注册
	registerJSON := `{
		"username": "testuser",
		"password": "password123",
		"email": "test@example.com"
	}`

	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(registerJSON))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	// 第二次注册相同用户名
	req2, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(registerJSON))
	req2.Header.Set("Content-Type", "application/json")

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}
