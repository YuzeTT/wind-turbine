package api

import (
	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
)

// ListTurbines 获取所有风机列表
// GET /api/v1/turbines?status=running
func ListTurbines(c *gin.Context) {
	status := c.Query("status")

	var turbines []model.Turbine
	query := db.DB.Order("id ASC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Find(&turbines)

	OK(c, turbines)
}

// GetTurbine 获取单台风机详情
// GET /api/v1/turbines/:id
func GetTurbine(c *gin.Context) {
	id := c.Param("id")

	var turbine model.Turbine
	if err := db.DB.First(&turbine, id).Error; err != nil {
		Fail(c, 404, "风机不存在")
		return
	}

	// 附加最近 10 条报警
	var alarms []model.Alarm
	db.DB.Where("turbine_id = ?", id).Order("created_at DESC").Limit(10).Find(&alarms)

	// 附加最近 7 天每日统计
	var stats []model.DailyStats
	db.DB.Where("turbine_id = ?", id).Order("date DESC").Limit(7).Find(&stats)

	OK(c, gin.H{
		"turbine":    turbine,
		"alarms":     alarms,
		"stats":      stats,
	})
}

// UpdateTurbineStatus 手动更新风机状态（手动启停等）
// PUT /api/v1/turbines/:id/status
func UpdateTurbineStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status   string `json:"status" binding:"required"`
		Operator string `json:"operator"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请求参数错误")
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{
		model.StatusRunning: true, model.StatusStandby: true,
		model.StatusMaintenance: true, model.StatusWeatherStop: true,
	}
	if !validStatuses[req.Status] {
		Fail(c, 400, "无效的状态值")
		return
	}
	// 不允许直接设为 fault（fault 只能通过报警流程）
	if req.Status == model.StatusFault {
		Fail(c, 400, "不能手动设为故障状态，请通过报警上报")
		return
	}

	if req.Operator == "" {
		req.Operator = "管理员"
	}

	var turbine model.Turbine
	if err := db.DB.First(&turbine, id).Error; err != nil {
		Fail(c, 404, "风机不存在")
		return
	}

	prevStatus := turbine.Status
	turbine.Status = req.Status
	if req.Status != model.StatusRunning {
		turbine.Power = 0
		turbine.RotorSpeed = 0
	}
	db.DB.Save(&turbine)

	// 记录操作日志
	db.DB.Create(&model.OperationLog{
		TurbineID:  turbine.ID,
		Operator:   req.Operator,
		Action:     "manual_" + req.Status,
		Reason:     req.Reason,
		PrevStatus: prevStatus,
		NewStatus:  req.Status,
	})

	// 系统日志
	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "api",
		Message: req.Operator + " 手动将 " + turbine.Name + " 从 " + prevStatus + " 改为 " + req.Status,
	})

	OK(c, turbine)
}

// ListTurbineModels 获取所有机型列表（筛选用）
// GET /api/v1/turbines/models
func ListTurbineModels(c *gin.Context) {
	var models []string
	db.DB.Model(&model.Turbine{}).Distinct("model").Pluck("model", &models)
	OK(c, models)
}
