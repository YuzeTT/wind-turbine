package api

import (
	"fmt"
	"time"

	"wind_turbine/backend/db"
	"wind_turbine/backend/model"

	"github.com/gin-gonic/gin"
)

// DashboardOverview 看板总览
// GET /api/v1/dashboard/overview
func DashboardOverview(c *gin.Context) {
	var turbines []model.Turbine
	db.DB.Find(&turbines)

	total := len(turbines)
	if total == 0 {
		OK(c, gin.H{})
		return
	}

	statusCount := map[string]int{}
	var totalPower, totalRatedPower float64
	var maxPower float64

	for _, t := range turbines {
		statusCount[t.Status]++
		totalPower += t.Power
		totalRatedPower += t.RatedPower
		if t.Power > maxPower {
			maxPower = t.Power
		}
	}

	// 今日总发电量
	var todayEnergy float64
	for _, t := range turbines {
		todayEnergy += t.TodayPower
	}

	// 活跃报警数
	var activeAlarms int64
	db.DB.Model(&model.Alarm{}).Where("status = ?", model.AlarmActive).Count(&activeAlarms)

	OK(c, gin.H{
		"total_turbines":      total,
		"running":             statusCount[model.StatusRunning],
		"fault":               statusCount[model.StatusFault],
		"maintenance":        statusCount[model.StatusMaintenance],
		"weather_stop":        statusCount[model.StatusWeatherStop],
		"standby":             statusCount[model.StatusStandby],
		"total_power":         totalPower,
		"total_rated_power":   totalRatedPower,
		"capacity_factor":     totalPower / totalRatedPower * 100, // 容量因子 %
		"max_power":           maxPower,
		"today_energy":       todayEnergy,
		"active_alarms":      activeAlarms,
		"online_ratio":       float64(statusCount[model.StatusRunning]) / float64(total) * 100,
	})
}

// StatusDistribution 状态分布（饼图）
// GET /api/v1/dashboard/status-distribution
func StatusDistribution(c *gin.Context) {
	var result []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	db.DB.Model(&model.Turbine{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&result)

	OK(c, result)
}

// PowerTrend 发电功率趋势（最近24小时，按小时聚合）
// GET /api/v1/dashboard/power-trend
func PowerTrend(c *gin.Context) {
	// 返回最近24小时每小时的模拟数据
	now := time.Now()
	hours := 24

	// 从每日统计里取最近7天，再补当天每小时模拟数据
	type hourData struct {
		Hour       string  `json:"hour"`
		TotalPower float64 `json:"total_power"`
		AvgPower   float64 `json:"avg_power"`
	}

	var result []hourData

	for i := hours - 1; i >= 0; i-- {
		h := now.Add(-time.Duration(i) * time.Hour)
		hourStr := h.Format("15:04")

		// 从当前风机数据模拟每小时数据
		var turbines []model.Turbine
		db.DB.Find(&turbines)

		var total float64
		for _, t := range turbines {
			if t.Status == model.StatusRunning {
				// 加点随机波动模拟历史数据
				total += t.Power * (0.7 + 0.3*float64(hours-i)/float64(hours))
			}
		}
		// 凌晨/夜间风力小
		hour := h.Hour()
		if hour < 6 || hour > 20 {
			total *= 0.5
		}

		result = append(result, hourData{
			Hour:       hourStr,
			TotalPower: total,
			AvgPower:   total / float64(len(turbines)),
		})
	}

	OK(c, result)
}

// AvailabilityData 可用性图表数据
// GET /api/v1/dashboard/availability?days=7
func AvailabilityData(c *gin.Context) {
	days := paramInt(c, "days", 7)
	if days < 1 || days > 90 {
		days = 7
	}

	dateStr := time.Now().AddDate(0, 0, -days+1).Format("2006-01-02")

	var stats []model.DailyStats
	db.DB.Where("date >= ?", dateStr).Order("date ASC, turbine_id ASC").Find(&stats)

	// 按日期聚合
	type dateAgg struct {
		Date          string  `json:"date"`
		AvgAvail      float64 `json:"avg_availability"`
		TotalEnergy   float64 `json:"total_energy"`
		AvgWindSpeed  float64 `json:"avg_wind_speed"`
		FaultCount    int     `json:"fault_count"`
		TurbineCount  int     `json:"turbine_count"`
	}

	aggMap := map[string]*dateAgg{}
	for _, s := range stats {
		agg, ok := aggMap[s.Date]
		if !ok {
			agg = &dateAgg{Date: s.Date}
			aggMap[s.Date] = agg
		}
		agg.AvgAvail += s.Availability
		agg.TotalEnergy += s.TotalPower
		agg.AvgWindSpeed += s.AvgWindSpeed
		agg.FaultCount += s.FaultCount
		agg.TurbineCount++
	}

	// 转成 slice 并计算平均
	var result []dateAgg
	for i := 0; i < days; i++ {
		d := time.Now().AddDate(0, 0, -days+1+i).Format("2006-01-02")
		if agg, ok := aggMap[d]; ok {
			if agg.TurbineCount > 0 {
				agg.AvgAvail /= float64(agg.TurbineCount)
				agg.AvgWindSpeed /= float64(agg.TurbineCount)
			}
			result = append(result, *agg)
		} else {
			result = append(result, dateAgg{
				Date:         d,
				AvgAvail:     0,
				TotalEnergy:  0,
				AvgWindSpeed: 0,
				FaultCount:   0,
				TurbineCount: 0,
			})
		}
	}

	OK(c, result)
}

// PowerByTurbine 各风机功率排行
// GET /api/v1/dashboard/power-ranking
func PowerByTurbine(c *gin.Context) {
	var turbines []model.Turbine
	db.DB.Order("power DESC").Find(&turbines)

	type item struct {
		Name       string  `json:"name"`
		Power      float64 `json:"power"`
		RatedPower float64 `json:"rated_power"`
		Status     string  `json:"status"`
		WindSpeed  float64 `json:"wind_speed"`
		RotorSpeed float64 `json:"rotor_speed"`
	}

	var result []item
	for _, t := range turbines {
		result = append(result, item{
			Name:       t.Name,
			Power:      t.Power,
			RatedPower: t.RatedPower,
			Status:     t.Status,
			WindSpeed:  t.WindSpeed,
			RotorSpeed: t.RotorSpeed,
		})
	}

	OK(c, result)
}

// WindRoseData 风玫瑰图数据（风向分布）
// GET /api/v1/dashboard/wind-rose
func WindRoseData(c *gin.Context) {
	var turbines []model.Turbine
	db.DB.Find(&turbines)

	// 16 个方位
	sectors := make([]struct {
		Sector    string  `json:"sector"`
		Angle     int     `json:"angle"`
		Count     int     `json:"count"`
		AvgSpeed  float64 `json:"avg_speed"`
	}, 16)

	sectorNames := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}

	for i := range sectors {
		sectors[i].Sector = sectorNames[i]
		sectors[i].Angle = i * 22 + 11
	}

	var totalSpeed [16]float64
	for _, t := range turbines {
		dir := t.WindDirection
		sector := int((dir + 11.25) / 22.5) % 16
		sectors[sector].Count++
		totalSpeed[sector] += t.WindSpeed
	}

	for i := range sectors {
		if sectors[i].Count > 0 {
			sectors[i].AvgSpeed = totalSpeed[i] / float64(sectors[i].Count)
		}
	}

	OK(c, sectors)
}

// DailyPowerTrend 最近 7 天每日总发电量趋势
// GET /api/v1/dashboard/daily-energy
func DailyPowerTrend(c *gin.Context) {
	type item struct {
		Date        string  `json:"date"`
		TotalEnergy float64 `json:"total_energy"`
	}

	var result []item
	sevenDaysAgo := time.Now().AddDate(0, 0, -6).Format("2006-01-02")

	rows := []struct {
		Date        string
		TotalEnergy float64
	}{}

	db.DB.Model(&model.DailyStats{}).
		Select("date, sum(total_power) as total_energy").
		Where("date >= ?", sevenDaysAgo).
		Group("date").
		Order("date ASC").
		Scan(&rows)

	for _, r := range rows {
		result = append(result, item{
			Date:        r.Date,
			TotalEnergy: r.TotalEnergy,
		})
	}

	// 补当天
	var todayEnergy float64
	db.DB.Model(&model.Turbine{}).Select("COALESCE(sum(today_power), 0)").Scan(&todayEnergy)
	result = append(result, item{
		Date:        time.Now().Format("2006-01-02"),
		TotalEnergy: todayEnergy,
	})

	OK(c, result)
}

// MapData 风机地图分布
// GET /api/v1/dashboard/map
func MapData(c *gin.Context) {
	var turbines []model.Turbine
	db.DB.Order("id ASC").Find(&turbines)

	type marker struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Status      string  `json:"status"`
		Power       float64 `json:"power"`
		RatedPower  float64 `json:"rated_power"`
		WindSpeed   float64 `json:"wind_speed"`
	}

	var result []marker
	for _, t := range turbines {
		result = append(result, marker{
			ID:         t.ID,
			Name:       t.Name,
			Latitude:   t.Latitude,
			Longitude:  t.Longitude,
			Status:     t.Status,
			Power:      t.Power,
			RatedPower: t.RatedPower,
			WindSpeed:  t.WindSpeed,
		})
	}

	OK(c, gin.H{
		"markers":  result,
		"center":   gin.H{"lat": 30.12, "lng": 122.85},
		"farm":     "东海风电场",
	})
}

// SystemInfo 系统信息
// GET /api/v1/dashboard/system-info
func SystemInfo(c *gin.Context) {
	var turbineCount, alarmCount, logCount int64
	db.DB.Model(&model.Turbine{}).Count(&turbineCount)
	db.DB.Model(&model.Alarm{}).Where("status = ?", model.AlarmActive).Count(&alarmCount)
	db.DB.Model(&model.OperationLog{}).Count(&logCount)

	var totalEnergy float64
	db.DB.Model(&model.Turbine{}).Select("COALESCE(sum(total_power), 0)").Scan(&totalEnergy)

	OK(c, gin.H{
		"turbine_count":   turbineCount,
		"active_alarms":   alarmCount,
		"operation_logs":  logCount,
		"total_energy_kwh": fmt.Sprintf("%.0f", totalEnergy),
		"system_time":     time.Now().Format("2006-01-02 15:04:05"),
	})
}
