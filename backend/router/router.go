package router

import (
	"wind_turbine/backend/api"
	"wind_turbine/backend/middleware"
	"wind_turbine/backend/ws"

	"github.com/gin-gonic/gin"
)

func Setup(hub *ws.Hub) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ─── 无需鉴权的路由 ───

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		api.OK(c, gin.H{"status": "ok"})
	})

	// 登录（无需 token）
	r.POST("/api/v1/auth/login", api.Login)

	// WebSocket（通过 query 参数鉴权）
	r.GET("/ws", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(401, gin.H{"code": 401, "message": "WebSocket 连接需要 token 参数"})
			return
		}
		claims, err := middleware.WSAuth(token)
		if err != nil {
			c.JSON(401, gin.H{"code": 401, "message": "token 无效或已过期"})
			return
		}
		c.Set("username", claims.Username)
		hub.HandleWS(c.Writer, c.Request)
	})

	// ─── 以下全部需要鉴权 ───
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthRequired())
	{
		// 认证相关（需要登录）
		auth := v1.Group("/auth")
		{
			auth.GET("/profile", api.GetProfile)
			auth.PUT("/password", api.ChangePassword)
			// 仅管理员
			adminOnly := auth.Group("")
			adminOnly.Use(middleware.AdminRequired())
			{
				adminOnly.POST("/register", api.Register)
				adminOnly.GET("/users", api.ListUsers)
				adminOnly.PUT("/users/:id/status", api.UpdateUserStatus)
			}
		}

		// 风机管理
		turbines := v1.Group("/turbines")
		{
			turbines.GET("", api.ListTurbines)
			turbines.GET("/models", api.ListTurbineModels)
			turbines.GET("/:id", api.GetTurbine)
			// 写操作需要管理员
			writeTurbines := turbines.Group("")
			writeTurbines.Use(middleware.AdminRequired())
			{
				writeTurbines.PUT("/:id/status", api.UpdateTurbineStatus)
			}
		}

		// 报警管理
		alarms := v1.Group("/alarms")
		{
			alarms.GET("", api.ListAlarms)
			alarms.GET("/stats", api.AlarmStats)
			// 写操作需要管理员
			writeAlarms := alarms.Group("")
			writeAlarms.Use(middleware.AdminRequired())
			{
				writeAlarms.POST("", api.CreateAlarm)
				writeAlarms.PUT("/:id/resolve", api.ResolveAlarm)
			}
		}

		// 操作日志
		oplogs := v1.Group("/operation-logs")
		{
			oplogs.GET("", api.ListOperationLogs)
			// 创建需要管理员
			writeOplogs := oplogs.Group("")
			writeOplogs.Use(middleware.AdminRequired())
			{
				writeOplogs.POST("", api.CreateOperationLog)
			}
		}

		// 系统日志
		v1.GET("/system-logs", api.ListSystemLogs)

		// 看板（只需登录）
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/overview", api.DashboardOverview)
			dashboard.GET("/status-distribution", api.StatusDistribution)
			dashboard.GET("/power-trend", api.PowerTrend)
			dashboard.GET("/availability", api.AvailabilityData)
			dashboard.GET("/power-ranking", api.PowerByTurbine)
			dashboard.GET("/wind-rose", api.WindRoseData)
			dashboard.GET("/daily-energy", api.DailyPowerTrend)
			dashboard.GET("/map", api.MapData)
			dashboard.GET("/system-info", api.SystemInfo)
		}

		// WebSocket 状态
		v1.GET("/ws/status", api.WSStatus(hub))
	}

	return r
}
