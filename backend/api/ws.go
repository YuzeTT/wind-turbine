package api

import (
	"wind_turbine/backend/db"
	"wind_turbine/backend/model"
	"wind_turbine/backend/ws"

	"github.com/gin-gonic/gin"
)

// WSHub WebSocket 处理
func WSHub(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		hub.HandleWS(c.Writer, c.Request)
	}
}

// WSStatus 返回 WebSocket 连接状态
// GET /api/v1/ws/status
func WSStatus(hub *ws.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var turbines []model.Turbine
		db.DB.Find(&turbines)

		OK(c, gin.H{
			"online_clients": hub.ClientCount(),
			"turbines":       len(turbines),
			"ws_endpoint":    "/ws",
		})
	}
}
