package api

import (
	"net/http"

	"wind_turbine/backend/db"
	"wind_turbine/backend/middleware"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Login 登录
// POST /api/v1/auth/login
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请输入用户名和密码")
		return
	}

	var user model.User
	if err := db.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		Fail(c, 401, "用户名或密码错误")
		return
	}

	if user.Status != "active" {
		Fail(c, 403, "账号已被禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		Fail(c, 401, "用户名或密码错误")
		return
	}

	token, err := middleware.GenerateToken(&user)
	if err != nil {
		Fail(c, 500, "生成令牌失败")
		return
	}

	// 记录系统日志
	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "auth",
		Message: user.Nickname + "(" + user.Username + ") 登录系统",
	})

	OK(c, gin.H{
		"token":    token,
		"expires":  86400, // 24h in seconds
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"role":     user.Role,
		},
	})
}

// Register 注册（仅管理员可创建新用户）
// POST /api/v1/auth/register
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Nickname string `json:"nickname"`
		Role     string `json:"role"` // admin / operator
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请填写完整的注册信息")
		return
	}

	if len(req.Password) < 6 {
		Fail(c, 400, "密码长度不能少于 6 位")
		return
	}

	if req.Role != "admin" && req.Role != "operator" {
		req.Role = "operator"
	}
	if req.Nickname == "" {
		req.Nickname = req.Username
	}

	// 检查用户名是否已存在
	var count int64
	db.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count)
	if count > 0 {
		Fail(c, 409, "用户名已存在")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		Fail(c, 500, "密码加密失败")
		return
	}

	user := model.User{
		Username: req.Username,
		Password: string(hashed),
		Nickname: req.Nickname,
		Role:     req.Role,
		Status:   "active",
	}
	db.DB.Create(&user)

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "注册成功",
		Data: gin.H{
			"id":       user.ID,
			"username": user.Username,
			"nickname": user.Nickname,
			"role":     user.Role,
		},
	})
}

// GetProfile 获取当前登录用户信息
// GET /api/v1/auth/profile
func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	nickname, _ := c.Get("nickname")
	role, _ := c.Get("role")

	OK(c, gin.H{
		"user_id":  userID,
		"username": username,
		"nickname": nickname,
		"role":     role,
	})
}

// ChangePassword 修改密码
// PUT /api/v1/auth/password
func ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请填写旧密码和新密码")
		return
	}

	if len(req.NewPassword) < 6 {
		Fail(c, 400, "新密码长度不能少于 6 位")
		return
	}

	userID, _ := c.Get("user_id")

	var user model.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		Fail(c, 404, "用户不存在")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		Fail(c, 401, "旧密码错误")
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	user.Password = string(hashed)
	db.DB.Save(&user)

	OKMsg(c, "密码修改成功")
}

// ListUsers 用户列表（仅管理员）
// GET /api/v1/auth/users
func ListUsers(c *gin.Context) {
	var users []model.User
	db.DB.Order("id ASC").Find(&users)
	OK(c, users)
}

// UpdateUserStatus 启用/禁用用户（仅管理员）
// PUT /api/v1/auth/users/:id/status
func UpdateUserStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"` // active / disabled
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请求参数错误")
		return
	}

	if req.Status != "active" && req.Status != "disabled" {
		Fail(c, 400, "无效状态值")
		return
	}

	var user model.User
	if err := db.DB.First(&user, id).Error; err != nil {
		Fail(c, 404, "用户不存在")
		return
	}

	user.Status = req.Status
	db.DB.Save(&user)
	OK(c, user)
}
