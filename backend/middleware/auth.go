package middleware

import (
	"net/http"
	"strings"
	"time"

	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWT 签名密钥（演示项目，生产环境应从配置读取）
var jwtSecret = []byte("wind-turbine-2026-secret-key")

// Claims JWT 载荷
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(user *model.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wind-turbine",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析 JWT Token
func ParseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// extractToken 从请求头提取 Bearer Token
func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return c.Query("token") // WebSocket 或降级到 query 参数
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

// AuthRequired 必须登录
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未提供认证令牌",
			})
			return
		}

		claims, err := ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "令牌无效或已过期",
			})
			return
		}

		// 查询用户确认仍然有效
		var user model.User
		if err := db.DB.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "用户不存在",
			})
			return
		}
		if user.Status != "active" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "账号已被禁用",
			})
			return
		}

		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("nickname", user.Nickname)
		c.Set("role", user.Role)
		c.Next()
	}
}

// AdminRequired 必须管理员
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "需要管理员权限",
			})
			return
		}
		c.Next()
	}
}

// WSAuth WebSocket 鉴权（通过 query 参数 token）
func WSAuth(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, ErrEmptyToken
	}
	return ParseToken(tokenStr)
}

var ErrEmptyToken = &tokenError{"token 为空"}

type tokenError struct{ msg string }

func (e *tokenError) Error() string { return e.msg }
