package api

import (
	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
)

// ListOperationLogs 操作日志列表
// GET /api/v1/operation-logs?turbine_id=1&action=maintenance&page=1&page_size=20
func ListOperationLogs(c *gin.Context) {
	page := paramInt(c, "page", 1)
	pageSize := paramInt(c, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := db.DB.Model(&model.OperationLog{})

	if tid := c.Query("turbine_id"); tid != "" {
		query = query.Where("turbine_id = ?", tid)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}

	var total int64
	query.Count(&total)

	var logs []model.OperationLog
	query.Preload("Turbine").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs)

	OKPage(c, logs, total, page, pageSize)
}

// CreateOperationLog 手动创建操作日志（维修停机/天气停机等）
// POST /api/v1/operation-logs
func CreateOperationLog(c *gin.Context) {
	var req struct {
		TurbineID uint   `json:"turbine_id" binding:"required"`
		Operator  string `json:"operator"`
		Action    string `json:"action" binding:"required"`
		Reason    string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请求参数错误")
		return
	}

	if req.Operator == "" {
		req.Operator = "管理员"
	}

	// 验证操作类型
	validActions := map[string]string{
		model.ActionMaintenance: model.StatusMaintenance,
		model.ActionWeatherStop: model.StatusWeatherStop,
		model.ActionManualStop:  model.StatusStandby,
		model.ActionManualStart: model.StatusRunning,
		model.ActionRestart:     model.StatusRunning,
	}
	newStatus, ok := validActions[req.Action]
	if !ok {
		Fail(c, 400, "无效的操作类型")
		return
	}

	var turbine model.Turbine
	if err := db.DB.First(&turbine, req.TurbineID).Error; err != nil {
		Fail(c, 404, "风机不存在")
		return
	}

	prevStatus := turbine.Status
	turbine.Status = newStatus
	if newStatus != model.StatusRunning {
		turbine.Power = 0
		turbine.RotorSpeed = 0
	}
	db.DB.Save(&turbine)

	ol := model.OperationLog{
		TurbineID:  req.TurbineID,
		Operator:   req.Operator,
		Action:     req.Action,
		Reason:     req.Reason,
		PrevStatus: prevStatus,
		NewStatus:  newStatus,
	}
	db.DB.Create(&ol)

	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "api",
		Message: req.Operator + " 对 " + turbine.Name + " 执行 " + req.Action + "：" + req.Reason,
	})

	db.DB.First(&ol, ol.ID).Preload("Turbine")
	OK(c, ol)
}
