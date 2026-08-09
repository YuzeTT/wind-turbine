package sim

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"wind_turbine/backend/db"
	"wind_turbine/backend/model"
	"wind_turbine/backend/ws"
)

const (
	cutInWind  = 3.0
	ratedWind  = 12.0
	cutOutWind = 25.0
	tickInterval = 3 * time.Second // 数据刷新间隔
)

// Simulator 风机数据模拟器
type Simulator struct {
	rng              *rand.Rand
	hub              *ws.Hub
	faultResolveTimes map[uint]time.Time // turbine_id → 故障自动恢复时间
	displayPower     map[uint]float64    // turbine_id → 缓起缓停显示功率
}

func New(hub *ws.Hub) *Simulator {
	return &Simulator{
		rng:               rand.New(rand.NewSource(time.Now().UnixNano())),
		hub:               hub,
		faultResolveTimes: make(map[uint]time.Time),
		displayPower:      make(map[uint]float64),
	}
}

// Start 启动模拟器
func (s *Simulator) Start() {
	go s.tickLoop()
	log.Println("[SIM] 模拟器已启动，数据刷新间隔:", tickInterval)

	// 每日统计 goroutine（每小时检查一次跨天）
	go s.dailyStatsLoop()
}

func (s *Simulator) tickLoop() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.tick()
	}
}

func (s *Simulator) tick() {
	var turbines []model.Turbine
	db.DB.Find(&turbines)

	now := time.Now()

	for i := range turbines {
		t := &turbines[i]
		s.updateTurbine(t, now)
	}

	// 批量保存
	for i := range turbines {
		db.DB.Model(&model.Turbine{}).Where("id = ?", turbines[i].ID).Updates(map[string]interface{}{
			"status":          turbines[i].Status,
			"power":           turbines[i].Power,
			"rotor_speed":     turbines[i].RotorSpeed,
			"wind_speed":      turbines[i].WindSpeed,
			"wind_direction":  turbines[i].WindDirection,
			"temperature":     turbines[i].Temperature,
			"last_update":      turbines[i].LastUpdate,
			"today_power":     turbines[i].TodayPower,
			"total_power":     turbines[i].TotalPower,
			"availability":    turbines[i].Availability,
		})
	}

	// 通过 WebSocket 分批广播：每台风机独立推送，错开 80ms 避免瞬时大量数据
	timeStr := now.Format("2006-01-02 15:04:05")
	go func(batch []model.Turbine, ts string) {
		for i := range batch {
			s.hub.Broadcast(map[string]interface{}{
				"type":    "turbine_update",
				"time":    ts,
				"turbine": batch[i],
			})
			time.Sleep(80 * time.Millisecond)
		}
	}(turbines, timeStr)

	// 随机生成报警（每个 tick 约 3% 概率）
	if s.rng.Float64() < 0.03 {
		s.randomAlarm(turbines)
	}

	// 自动排障：检查故障风机是否到达恢复时间
	s.autoResolveFaults(now)

	// 维护自动恢复：维护状态无活跃报警后转为待机
	s.autoResolveMaintenance(now)
}

func (s *Simulator) updateTurbine(t *model.Turbine, now time.Time) {
	// ─── 风速随机游走 ───
	delta := (s.rng.Float64() - 0.5) * 2.0 // ±1.0 m/s 波动
	newWind := t.WindSpeed + delta
	// 限制在合理范围 0-30
	if newWind < 0 {
		newWind = 0
	}
	if newWind > 30 {
		newWind = 30
	}
	t.WindSpeed = newWind

	// ─── 风向缓慢变化 ───
	dirDelta := (s.rng.Float64() - 0.5) * 10.0
	t.WindDirection += dirDelta
	if t.WindDirection < 0 {
		t.WindDirection += 360
	}
	if t.WindDirection >= 360 {
		t.WindDirection -= 360
	}

	// ─── 根据状态计算功率和转速（缓起缓停） ───
	switch t.Status {
	case model.StatusRunning:
		if t.WindSpeed >= cutOutWind {
			s.changeStatus(t, model.StatusWeatherStop, "风速超过切出阈值，自动停机")
			s.rampDown(t)
		} else if t.WindSpeed < cutInWind {
			// 风速不足，缓降
			s.rampDown(t)
		} else {
			// 正常运行：缓起或维持，在目标功率附近小幅浮动
			target := calcPower(t.WindSpeed, t.RatedPower)
			s.rampUp(t, target)
			t.RotorSpeed = calcRotorSpeed(t.WindSpeed)
		}

	case model.StatusStandby:
		s.rampDown(t)
		if t.WindSpeed >= cutInWind && t.WindSpeed < cutOutWind {
			t.Status = model.StatusRunning
		}

	case model.StatusWeatherStop:
		s.rampDown(t)
		if t.WindSpeed < cutOutWind-2 {
			t.Status = model.StatusStandby
		}

	case model.StatusFault, model.StatusMaintenance:
		s.rampDown(t)
	}

	// ─── 机舱温度 ───
	if t.Power > 0 {
		// 运行时温度偏高，随风速和功率变化
		targetTemp := 35.0 + (t.Power/t.RatedPower)*20.0
		t.Temperature += (targetTemp - t.Temperature) * 0.1
	} else {
		// 停机时温度逐渐降低
		t.Temperature += (25.0 - t.Temperature) * 0.05
	}
	// 温度随机波动
	t.Temperature += (s.rng.Float64() - 0.5) * 1.0

	// ─── 累加发电量 ───
	if t.Power > 0 {
		// tickInterval 秒的发电量 = power * (3/3600) kWh
		energyDelta := t.Power * (3.0 / 3600.0)
		t.TodayPower += energyDelta
		t.TotalPower += energyDelta
	}

	t.LastUpdate = now

	// ─── 随机故障（约 0.5% 概率每 tick） ───
	if t.Status == model.StatusRunning && s.rng.Float64() < 0.005 {
		s.triggerFault(t)
	}
}

// rampUp 缓起：3% 步长追赶目标功率，到达后在目标附近小幅浮动
func (s *Simulator) rampUp(t *model.Turbine, target float64) {
	current := s.displayPower[t.ID]
	diff := target - current

	if diff > -1 && diff < 1 {
		// 已到达目标：加微小浮动（±0.25%随机游走，钳位±2%）
		drift := (s.rng.Float64() - 0.5) * target * 0.005
		val := current + drift
		if val > target*1.02 {
			val = target * 1.02
		}
		if val < target*0.98 {
			val = target * 0.98
		}
		s.displayPower[t.ID] = val
	} else {
		// 3% 追赶
		s.displayPower[t.ID] = current + diff*0.03
	}
	t.Power = s.displayPower[t.ID]
}

// rampDown 缓停：3% 步长降到 0
func (s *Simulator) rampDown(t *model.Turbine) {
	current := s.displayPower[t.ID]
	if current > 1 {
		s.displayPower[t.ID] = current * 0.97
	} else {
		s.displayPower[t.ID] = 0
	}
	t.Power = s.displayPower[t.ID]
}

// changeStatus 改变风机状态并记录日志
func (s *Simulator) changeStatus(t *model.Turbine, newStatus, reason string) {
	prev := t.Status
	t.Status = newStatus

	db.DB.Create(&model.OperationLog{
		TurbineID:  t.ID,
		Operator:   "系统自动",
		Action:     "auto_" + newStatus,
		Reason:     reason,
		PrevStatus: prev,
		NewStatus:  newStatus,
	})

	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "simulator",
		Message: fmt.Sprintf("%s 状态变更: %s → %s（%s）", t.Name, prev, newStatus, reason),
	})
}

// autoResolveFaults 自动排障：故障持续 30-120 秒后自行恢复
func (s *Simulator) autoResolveFaults(now time.Time) {
	// 查询所有故障状态的风机
	var faultedTurbines []model.Turbine
	db.DB.Where("status = ?", model.StatusFault).Find(&faultedTurbines)

	for _, t := range faultedTurbines {
		// 首次发现故障：分配恢复时间（30-120 秒后）
		if _, exists := s.faultResolveTimes[t.ID]; !exists {
			s.faultResolveTimes[t.ID] = now.Add(time.Duration(30+s.rng.Intn(90)) * time.Second)
			continue
		}

		// 还没到恢复时间
		if now.Before(s.faultResolveTimes[t.ID]) {
			continue
		}

		// ─── 执行自动排障 ───

		// 1. 处理该风机最新的活跃报警
		var alarm model.Alarm
		if err := db.DB.Where("turbine_id = ? AND status = ?", t.ID, model.AlarmActive).
			Order("created_at DESC").First(&alarm).Error; err == nil {
			resolvedAt := now
			alarm.Status = model.AlarmResolved
			alarm.ResolvedAt = &resolvedAt
			alarm.ResolvedBy = "自动排障"
			db.DB.Save(&alarm)
		}

		// 2. 恢复为待机状态，等待手动启动
		newStatus := model.StatusStandby
		t.Status = newStatus
		db.DB.Save(&t)

		// 3. 操作日志
		db.DB.Create(&model.OperationLog{
			TurbineID:  t.ID,
			Operator:   "自动排障",
			Action:     model.ActionRestart,
			Reason:     "故障排除，转为待机等待启动",
			PrevStatus: model.StatusFault,
			NewStatus:  newStatus,
		})

		// 4. 系统日志
		db.DB.Create(&model.SystemLog{
			Level:   "info",
			Module:  "simulator",
			Message: fmt.Sprintf("%s 故障已自动排除，转为待机等待启动", t.Name),
		})

		delete(s.faultResolveTimes, t.ID)
		log.Printf("[SIM] %s 故障自动排除，恢复为 %s", t.Name, newStatus)
	}

	// 清理已不在故障状态的风机记录（如手动恢复的）
	for id := range s.faultResolveTimes {
		var count int64
		db.DB.Model(&model.Turbine{}).Where("id = ? AND status = ?", id, model.StatusFault).Count(&count)
		if count == 0 {
			delete(s.faultResolveTimes, id)
		}
	}
}

// autoResolveMaintenance 维护状态自动恢复：无活跃报警后等 10-20 秒转为待机
func (s *Simulator) autoResolveMaintenance(now time.Time) {
	var maintenanceTurbines []model.Turbine
	db.DB.Where("status = ?", model.StatusMaintenance).Find(&maintenanceTurbines)

	for _, t := range maintenanceTurbines {
		// 有活跃报警 → 等报警处理完
		var activeCount int64
		db.DB.Model(&model.Alarm{}).Where("turbine_id = ? AND status = ?", t.ID, model.AlarmActive).Count(&activeCount)
		if activeCount > 0 {
			// 还有未处理报警，不恢复，清理可能的旧计时
			delete(s.faultResolveTimes, t.ID)
			continue
		}

		// 无活跃报警：分配恢复时间（10-20 秒后转待机）
		if _, exists := s.faultResolveTimes[t.ID]; !exists {
			s.faultResolveTimes[t.ID] = now.Add(time.Duration(10+s.rng.Intn(10)) * time.Second)
			continue
		}

		if now.Before(s.faultResolveTimes[t.ID]) {
			continue
		}

		// 时间到 → 转为待机
		t.Status = model.StatusStandby
		db.DB.Save(&t)

		db.DB.Create(&model.OperationLog{
			TurbineID:  t.ID,
			Operator:   "系统自动",
			Action:     "auto_" + model.StatusStandby,
			Reason:     "维护完成，无活跃报警，转为待机等待启动",
			PrevStatus: model.StatusMaintenance,
			NewStatus:  model.StatusStandby,
		})

		db.DB.Create(&model.SystemLog{
			Level:   "info",
			Module:  "simulator",
			Message: fmt.Sprintf("%s 维护完成，转为待机等待启动", t.Name),
		})

		delete(s.faultResolveTimes, t.ID)
		log.Printf("[SIM] %s 维护完成，转为待机", t.Name)
	}

	// 清理已不在维护/故障状态的旧记录
	for id := range s.faultResolveTimes {
		var count int64
		db.DB.Model(&model.Turbine{}).Where("id = ? AND status IN ?", id, []string{model.StatusFault, model.StatusMaintenance}).Count(&count)
		if count == 0 {
			delete(s.faultResolveTimes, id)
		}
	}
}

func statusCN(status string) string {
	switch status {
	case model.StatusRunning:
		return "运行中"
	case model.StatusStandby:
		return "待机"
	case model.StatusMaintenance:
		return "维护"
	case model.StatusWeatherStop:
		return "天气停机"
	case model.StatusFault:
		return "故障"
	}
	return status
}

// triggerFault 触发随机故障
func (s *Simulator) triggerFault(t *model.Turbine) {
	faultTypes := []struct {
		typ, title, desc, severity string
	}{
		{model.AlarmHighTemp, "机舱温度过高", "机舱温度超过安全阈值，建议检查冷却系统", model.SeverityWarning},
		{model.AlarmVibration, "塔筒振动异常", "检测到异常振动信号，可能存在机械松动", model.SeverityWarning},
		{model.AlarmPitch, "变桨系统故障", "变桨角度偏差超出正常范围", model.SeverityCritical},
		{model.AlarmYaw, "偏航系统异常", "偏航角度偏离风向过大", model.SeverityWarning},
		{model.AlarmGearbox, "齿轮箱油温高", "齿轮箱润滑油温度偏高", model.SeverityWarning},
		{model.AlarmConverter, "变流器故障", "变流器输出异常，请检查功率模块", model.SeverityCritical},
	}

	ft := faultTypes[s.rng.Intn(len(faultTypes))]

	alarm := model.Alarm{
		TurbineID:   t.ID,
		Code:        fmt.Sprintf("A%04d", s.rng.Intn(9999)),
		Type:        ft.typ,
		Severity:    ft.severity,
		Title:       fmt.Sprintf("%s %s", t.Name, ft.title),
		Description: ft.desc,
		Status:      model.AlarmActive,
		Source:      "auto",
	}
	db.DB.Create(&alarm)

	// 严重故障直接停机
	if ft.severity == model.SeverityCritical {
		s.changeStatus(t, model.StatusFault, "触发严重故障: "+ft.title)
	}

	log.Printf("[SIM] %s 触发报警: %s", t.Name, ft.title)
}

// randomAlarm 随机选一台运行中的风机生成轻微报警
func (s *Simulator) randomAlarm(turbines []model.Turbine) {
	var running []model.Turbine
	for _, t := range turbines {
		if t.Status == model.StatusRunning {
			running = append(running, t)
		}
	}
	if len(running) == 0 {
		return
	}
	t := running[s.rng.Intn(len(running))]
	s.triggerFault(&t)
}

// ─── 每日统计 ───

func (s *Simulator) dailyStatsLoop() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.checkDailyStats()
	}
}

func (s *Simulator) checkDailyStats() {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	// 检查昨天的统计是否已生成
	var count int64
	db.DB.Model(&model.DailyStats{}).Where("date = ?", yesterday).Count(&count)
	if count > 0 {
		return
	}

	var turbines []model.Turbine
	db.DB.Find(&turbines)

	for _, t := range turbines {
		stats := model.DailyStats{
			TurbineID:     t.ID,
			Date:          yesterday,
			TotalPower:    t.TodayPower,
			MaxPower:      t.RatedPower * 0.9,
			MinPower:      0,
			AvgPower:      t.TodayPower / 24.0,
			AvgWindSpeed:  t.WindSpeed,
			Availability:  t.Availability,
			FaultCount:     0,
			RunHours:      20.0 + s.rng.Float64()*4.0,
		}
		db.DB.Create(&stats)
	}

	// 重置今日发电量
	db.DB.Model(&model.Turbine{}).Where("1=1").Update("today_power", 0)

	db.DB.Create(&model.SystemLog{
		Level:   "info",
		Module:  "simulator",
		Message: fmt.Sprintf("生成 %s 每日统计数据并重置日发电量", yesterday),
	})

	log.Printf("[SIM] 生成 %s 每日统计，重置日发电量", yesterday)
}

// ─── 功率曲线辅助函数 ───

func calcPower(windSpeed, ratedPower float64) float64 {
	switch {
	case windSpeed < cutInWind:
		return 0
	case windSpeed >= cutOutWind:
		return 0
	case windSpeed >= ratedWind:
		return ratedPower
	default:
		ratio := (windSpeed - cutInWind) / (ratedWind - cutInWind)
		return ratedPower * ratio * ratio * ratio
	}
}

func calcRotorSpeed(windSpeed float64) float64 {
	if windSpeed < cutInWind || windSpeed >= cutOutWind {
		return 0
	}
	if windSpeed >= ratedWind {
		return 15.0
	}
	ratio := (windSpeed - cutInWind) / (ratedWind - cutInWind)
	return 5.0 + 10.0*ratio
}
