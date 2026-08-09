package api

import (
	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
)

// ListSystemLogs 系统日志列表
// GET /api/v1/system-logs?level=error&module=simulator&page=1&page_size=20
func ListSystemLogs(c *gin.Context) {
	page := paramInt(c, "page", 1)
	pageSize := paramInt(c, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := db.DB.Model(&model.SystemLog{})

	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}

	var total int64
	query.Count(&total)

	var logs []model.SystemLog
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	OKPage(c, logs, total, page, pageSize)
}
