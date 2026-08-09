package db

import (
	"fmt"
	"math/rand"
	"time"

	"wind_turbine/backend/model"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Init 初始化数据库连接，autoMigrate 并生成种子数据
func Init(dbPath string) error {
	// 使用 modernc.org/sqlite 纯 Go 驱动，无需 CGO
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	DB = db

	if err := autoMigrate(db); err != nil {
		return fmt.Errorf("建表失败: %w", err)
	}

	if err := seed(db); err != nil {
		return fmt.Errorf("生成种子数据失败: %w", err)
	}

	return nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Turbine{},
		&model.Alarm{},
		&model.OperationLog{},
		&model.SystemLog{},
		&model.DailyStats{},
		&model.User{},
	)
}

func seed(db *gorm.DB) error {
	// 只在表为空时生成种子数据
	var count int64
	db.Model(&model.Turbine{}).Count(&count)
	if count > 0 {
		return nil
	}

	// ─── 30 台风机 ───
	farm := "东海风电场"
	turbineModels := []struct {
		model      string
		ratedPower float64
	}{
		{"Goldwind GW155-3.4MW", 3400},
		{"Goldwind GW136-2.5MW", 2500},
		{"Envision EN-156-4.5MW", 4500},
		{"Mingyang MySE3.0-135",  3000},
	}

	rng := rand.New(rand.NewSource(42))
	now := time.Now()
	baseLat := 30.12  // 风场中心纬度
	baseLng := 122.85 // 风场中心经度

	for i := 1; i <= 30; i++ {
		tm := turbineModels[rng.Intn(len(turbineModels))]
		// 随机风速初始值 4-18 m/s
		windSpeed := 4.0 + rng.Float64()*14.0
		status := model.StatusRunning
		power := calcPower(windSpeed, tm.ratedPower)
		rotorSpeed := calcRotorSpeed(windSpeed, power > 0)

		// 约 10% 概率处于非运行状态
		switch rng.Intn(10) {
		case 0:
			status = model.StatusStandby
			power = 0
			rotorSpeed = 0
		case 1:
			status = model.StatusMaintenance
			power = 0
			rotorSpeed = 0
		}

		// 随机偏移坐标，模拟排布
		lat := baseLat + (rng.Float64()-0.5)*0.04
		lng := baseLng + (rng.Float64()-0.5)*0.06

		turbine := model.Turbine{
			Name:          fmt.Sprintf("WT-%03d", i),
			Model:         tm.model,
			RatedPower:    tm.ratedPower,
			Status:        status,
			Power:         power,
			RotorSpeed:    rotorSpeed,
			WindSpeed:     windSpeed,
			WindDirection: 30.0 + rng.Float64()*60.0,
			Temperature:   25.0 + rng.Float64()*15.0,
			Latitude:      lat,
			Longitude:     lng,
			InstalledDate: now.AddDate(-rng.Intn(3), -rng.Intn(12), -rng.Intn(28)),
			LastUpdate:    now,
			TodayPower:    power * rng.Float64() * 2, // 模拟已运行几小时
			TotalPower:    rng.Float64() * 5_000_000, // 累计发电量
			Availability:  95.0 + rng.Float64()*4.9,
			Farm:          farm,
		}
		db.Create(&turbine)
	}

	// ─── 生成 7 天历史每日统计 ───
	var turbines []model.Turbine
	db.Find(&turbines)

	for day := 6; day >= 0; day-- {
		dateStr := now.AddDate(0, 0, -day).Format("2006-01-02")
		for _, t := range turbines {
			avgWind := 5.0 + rng.Float64()*10.0
			maxPower := calcPower(avgWind+3, t.RatedPower)
			minPower := calcPower(2.0, t.RatedPower)
			if minPower > maxPower {
				minPower = maxPower * 0.2
			}
			// 运行小时 18-24h
			runHours := 18.0 + rng.Float64()*6.0
			totalPower := maxPower * runHours * 0.7 // 平均效率
			availability := runHours / 24.0 * 100.0

			db.Create(&model.DailyStats{
				TurbineID:     t.ID,
				Turbine:       nil,
				Date:          dateStr,
				TotalPower:    totalPower,
				MaxPower:      maxPower,
				MinPower:      minPower,
				AvgPower:      totalPower / runHours,
				AvgWindSpeed:  avgWind,
				Availability:  availability,
				FaultCount:     rng.Intn(3),
				RunHours:      runHours,
			})
		}
	}

	// ─── 一些初始报警 ───
	for _, t := range turbines[:3] {
		alarmTime := now.Add(-time.Duration(rng.Intn(48)) * time.Hour)
		db.Create(&model.Alarm{
			TurbineID:   t.ID,
			Turbine:     nil,
			Code:        fmt.Sprintf("A%04d", rng.Intn(9999)),
			Type:        []string{model.AlarmHighTemp, model.AlarmVibration, model.AlarmPitch}[rng.Intn(3)],
			Severity:    []string{model.SeverityInfo, model.SeverityWarning, model.SeverityCritical}[rng.Intn(3)],
			Title:       fmt.Sprintf("%s 触发报警", t.Name),
			Description: "系统检测到参数异常，请检查设备状态",
			Status:      model.AlarmActive,
			Source:      "auto",
			CreatedAt:   alarmTime,
		})
	}

	// ─── 一些初始操作日志 ───
	actions := []struct {
		action, reason string
	}{
		{model.ActionMaintenance, "定期检修"},
		{model.ActionManualStop, "电网调度要求降负荷"},
		{model.ActionWeatherStop, "台风预警，主动停机"},
		{model.ActionRestart, "故障排除后重启"},
	}
	for i, t := range turbines[:4] {
		a := actions[i]
		ol := model.OperationLog{
			TurbineID:  t.ID,
			Turbine:    nil,
			Operator:   []string{"张三", "李四", "王五", "赵六"}[i],
			Action:     a.action,
			Reason:     a.reason,
			PrevStatus: model.StatusRunning,
			NewStatus:  model.StatusMaintenance,
			CreatedAt:  now.Add(-time.Duration(rng.Intn(72)) * time.Hour),
		}
		db.Create(&ol)
	}

	// ─── 一些初始系统日志 ───
	sysLogs := []model.SystemLog{
		{Level: "info", Module: "system", Message: "系统启动完成，风机数据同步中"},
		{Level: "info", Module: "simulator", Message: "数据模拟器已启动"},
		{Level: "warning", Module: "alarm", Message: "检测到 WT-001 机舱温度偏高"},
		{Level: "info", Module: "api", Message: "API 服务就绪"},
	}
	for _, sl := range sysLogs {
		sl.CreatedAt = now.Add(-time.Duration(rng.Intn(24)) * time.Hour)
		db.Create(&sl)
	}

	// ─── 默认管理员账户 ───
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount == 0 {
		// bcrypt hash for "admin123"
		hashed, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username: "admin",
			Password: string(hashed),
			Nickname: "系统管理员",
			Role:     "admin",
			Status:   "active",
		})
		// 普通操作员
		hashed2, _ := bcrypt.GenerateFromPassword([]byte("op123456"), bcrypt.DefaultCost)
		db.Create(&model.User{
			Username: "operator",
			Password: string(hashed2),
			Nickname: "值班员",
			Role:     "operator",
			Status:   "active",
		})
	}

	return nil
}

// ─── 功率曲线辅助函数 ───

const (
	cutInWind  = 3.0  // 切入风速 m/s
	ratedWind  = 12.0 // 额定风速 m/s
	cutOutWind = 25.0 // 切出风速 m/s
)

// calcPower 根据风速计算功率 (简化功率曲线)
func calcPower(windSpeed, ratedPower float64) float64 {
	switch {
	case windSpeed < cutInWind:
		return 0
	case windSpeed >= cutOutWind:
		return 0 // 切出
	case windSpeed >= ratedWind:
		return ratedPower // 额定功率
	default:
		// 立方关系近似: P = P_rated * ((v - v_in) / (v_rated - v_in))^3
		ratio := (windSpeed - cutInWind) / (ratedWind - cutInWind)
		return ratedPower * ratio * ratio * ratio
	}
}

// calcRotorSpeed 根据风速计算转速
func calcRotorSpeed(windSpeed float64, running bool) float64 {
	if !running {
		return 0
	}
	if windSpeed < cutInWind {
		return 0
	}
	if windSpeed >= cutOutWind {
		return 0
	}
	// 5-15 rpm，随风速线性增长
	minRPM := 5.0
	maxRPM := 15.0
	if windSpeed >= ratedWind {
		return maxRPM
	}
	ratio := (windSpeed - cutInWind) / (ratedWind - cutInWind)
	return minRPM + (maxRPM-minRPM)*ratio
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
