package api

import (
	"fmt"
	"time"

	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
)

// ListAlarms 报警列表
// GET /api/v1/alarms?status=active&severity=critical&turbine_id=1&page=1&page_size=20
func ListAlarms(c *gin.Context) {
	page := paramInt(c, "page", 1)
	pageSize := paramInt(c, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := db.DB.Model(&model.Alarm{})

	if s := c.Query("status"); s != "" {
		query = query.Where("status = ?", s)
	}
	if s := c.Query("severity"); s != "" {
		query = query.Where("severity = ?", s)
	}
	if tid := c.Param("turbine_id"); tid != "" {
		query = query.Where("turbine_id = ?", tid)
	} else if tid := c.Query("turbine_id"); tid != "" {
		query = query.Where("turbine_id = ?", tid)
	}

	var total int64
	query.Count(&total)

	var alarms []model.Alarm
	query.Preload("Turbine").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&alarms)

	OKPage(c, alarms, total, page, pageSize)
}

// CreateAlarm 手动上报故障
// POST /api/v1/alarms
func CreateAlarm(c *gin.Context) {
	var req struct {
		TurbineID   uint   `json:"turbine_id" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Severity    string `json:"severity" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Operator    string `json:"operator"`
		StopTurbine bool   `json:"stop_turbine"` // 是否同时停机
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, "请求参数错误")
		return
	}

	// 验证风机存在
	var turbine model.Turbine
	if err := db.DB.First(&turbine, req.TurbineID).Error; err != nil {
		Fail(c, 404, "风机不存在")
		return
	}

	if req.Operator == "" {
		req.Operator = "管理员"
	}

	alarm := model.Alarm{
		TurbineID:   req.TurbineID,
		Code:        fmt.Sprintf("M%04d", time.Now().Unix()%10000),
		Type:        req.Type,
		Severity:    req.Severity,
		Title:       fmt.Sprintf("%s %s", turbine.Name, req.Title),
		Description: req.Description,
		Status:      model.AlarmActive,
		Source:      "manual",
	}
	if err := db.DB.Create(&alarm).Error; err != nil {
		Fail(c, 500, "创建报警失败")
		return
	}

	// 操作日志
	prevStatus := turbine.Status
	action := model.ActionFaultReport
	newStatus := turbine.Status

	if req.StopTurbine || req.Severity == model.SeverityCritical {
		turbine.Status = model.StatusFault
		turbine.Power = 0
		turbine.RotorSpeed = 0
		db.DB.Save(&turbine)
		newStatus = model.StatusFault
		action = model.ActionFaultReport
	}

	db.DB.Create(&model.OperationLog{
		TurbineID:  req.TurbineID,
		Operator:   req.Operator,
		Action:     action,
		Reason:     req.Title,
		PrevStatus: prevStatus,
		NewStatus:  newStatus,
	})

	db.DB.First(&alarm, alarm.ID).Preload("Turbine")
	OK(c, alarm)
}

// ResolveAlarm 处理报警
// PUT /api/v1/alarms/:id/resolve
func ResolveAlarm(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ResolvedBy string `json:"resolved_by"`
		Comment    string `json:"comment"`
		Restart    bool   `json:"restart"` // 是否恢复运行
	}
	c.ShouldBindJSON(&req)

	if req.ResolvedBy == "" {
		req.ResolvedBy = "管理员"
	}

	var alarm model.Alarm
	if err := db.DB.First(&alarm, id).Error; err != nil {
		Fail(c, 404, "报警不存在")
		return
	}

	now := time.Now()
	alarm.Status = model.AlarmResolved
	alarm.ResolvedAt = &now
	alarm.ResolvedBy = req.ResolvedBy
	db.DB.Save(&alarm)

	// 如果需要恢复运行
	if req.Restart {
		var turbine model.Turbine
		db.DB.First(&turbine, alarm.TurbineID)
		prev := turbine.Status
		turbine.Status = model.StatusRunning
		db.DB.Save(&turbine)

		db.DB.Create(&model.OperationLog{
			TurbineID:  turbine.ID,
			Operator:   req.ResolvedBy,
			Action:     model.ActionRestart,
			Reason:     "报警处理后重启 - " + req.Comment,
			PrevStatus: prev,
			NewStatus:  model.StatusRunning,
		})
	}

	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "api",
		Message: req.ResolvedBy + " 处理了报警 #" + id + " " + alarm.Title,
	})

	db.DB.First(&alarm, id).Preload("Turbine")
	OK(c, alarm)
}

// AlarmStats 报警统计
// GET /api/v1/alarms/stats
func AlarmStats(c *gin.Context) {
	type Stat struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}

	var severityStats []Stat
	db.DB.Model(&model.Alarm{}).
		Select("severity, count(*) as count").
		Group("severity").
		Scan(&severityStats)

	var activeCount int64
	db.DB.Model(&model.Alarm{}).Where("status = ?", model.AlarmActive).Count(&activeCount)

	var todayCount int64
	db.DB.Model(&model.Alarm{}).Where("date(created_at) = date('now')").Count(&todayCount)

	OK(c, gin.H{
		"by_severity": severityStats,
		"active":      activeCount,
		"today":       todayCount,
	})
}
