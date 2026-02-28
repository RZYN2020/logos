// Package handlers 认证处理器
package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/log-system/log-analyzer/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authConfig *middleware.AuthConfig
	users      map[string]*User // 内存存储，生产环境应使用数据库
}

// User 用户模型
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"`
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authConfig *middleware.AuthConfig) *AuthHandler {
	h := &AuthHandler{
		authConfig: authConfig,
		users:      make(map[string]*User),
	}

	// 创建默认管理员用户
	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	h.users["admin"] = &User{
		ID:        "admin-001",
		Username:  "admin",
		Password:  string(hashedPwd),
		Email:     "admin@logos.com",
		Roles:     []string{"admin", "user"},
		CreatedAt: time.Now(),
	}

	return h
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := h.users[req.Username]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid username or password"})
		return
	}

	token, err := h.authConfig.GenerateToken(user.ID, user.Username, user.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"roles":    user.Roles,
		},
	})
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查用户是否已存在
	if _, exists := h.users[req.Username]; exists {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	// 密码加密
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Password:  string(hashedPwd),
		Email:     req.Email,
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
	}

	h.users[req.Username] = user

	token, err := h.authConfig.GenerateToken(user.ID, user.Username, user.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"roles":    user.Roles,
		},
	})
}

// GetCurrentUser 获取当前用户信息
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	roles, _ := c.Get("roles")

	c.JSON(http.StatusOK, gin.H{
		"user_id":  userID,
		"username": username,
		"roles":    roles,
	})
}

// ChangePassword 修改密码
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	username, _ := c.Get("username")

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, exists := h.users[username.(string)]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid old password"})
		return
	}

	// 更新密码
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user.Password = string(hashedPwd)
	h.users[username.(string)] = user

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// ListUsers 获取用户列表（需要管理员权限）
func (h *AuthHandler) ListUsers(c *gin.Context) {
	var users []gin.H
	for _, user := range h.users {
		users = append(users, gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"roles":     user.Roles,
			"created_at": user.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}
