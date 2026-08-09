package model

import (
	"time"

	"gorm.io/gorm"
)

// ─── 风机状态常量 ───
const (
	StatusRunning      = "running"       // 正常运行
	StatusFault        = "fault"         // 故障
	StatusMaintenance  = "maintenance"   // 维护
	StatusWeatherStop  = "weather_stop"  // 天气停机
	StatusStandby      = "standby"        // 待机
)

// ─── 报警级别 ───
const (
	SeverityInfo     = "info"     // 信息
	SeverityWarning  = "warning"  // 警告
	SeverityCritical = "critical"  // 严重
)

// ─── 报警状态 ───
const (
	AlarmActive   = "active"   // 活跃
	AlarmResolved = "resolved" // 已处理
)

// ─── 报警类型 ───
const (
	AlarmHighTemp    = "high_temp"    // 高温
	AlarmVibration   = "vibration"    // 振动
	AlarmPitch       = "pitch"        // 变桨
	AlarmYaw         = "yaw"          // 偏航
	AlarmGrid        = "grid"         // 电网
	AlarmConverter   = "converter"    // 变流器
	AlarmGearbox     = "gearbox"      // 齿轮箱
	AlarmWindSensor  = "wind_sensor"  // 风传感器
)

// ─── 操作类型 ───
const (
	ActionFaultReport   = "fault_report"    // 故障上报
	ActionMaintenance   = "maintenance"     // 维修停机
	ActionWeatherStop   = "weather_stop"     // 天气停机
	ActionManualStop    = "manual_stop"     // 手动停机
	ActionManualStart   = "manual_start"    // 手动启动
	ActionRestart       = "restart"         // 重启
)

// Turbine 风力发电机
type Turbine struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name"`           // 风机编号 WT-001
	Model         string    `json:"model"`          // 机型
	RatedPower    float64   `json:"rated_power"`    // 额定功率 kW
	Status        string    `json:"status"`         // 当前状态
	Power         float64   `json:"power"`          // 当前功率 kW
	RotorSpeed    float64   `json:"rotor_speed"`    // 转速 rpm
	WindSpeed     float64   `json:"wind_speed"`     // 风速 m/s
	WindDirection float64   `json:"wind_direction"` // 风向 °
	Temperature   float64   `json:"temperature"`    // 机舱温度 ℃
	Latitude      float64   `json:"latitude"`       // 纬度
	Longitude     float64   `json:"longitude"`      // 经度
	InstalledDate time.Time `json:"installed_date"` // 投运日期
	LastUpdate    time.Time `json:"last_update"`    // 最后更新时间
	TodayPower    float64   `json:"today_power"`    // 今日发电量 kWh
	TotalPower    float64   `json:"total_power"`    // 累计发电量 kWh
	Availability  float64   `json:"availability"`   // 可用率 %
	Farm          string    `json:"farm"`            // 风场名称
}

func (Turbine) TableName() string { return "turbines" }

// Alarm 报警
type Alarm struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	TurbineID   uint       `json:"turbine_id"`
	Turbine     *Turbine   `json:"turbine,omitempty" gorm:"foreignKey:TurbineID"`
	Code        string     `json:"code"`            // 报警代码
	Type        string     `json:"type"`            // 类型
	Severity    string     `json:"severity"`        // 级别
	Title       string     `json:"title"`            // 标题
	Description string     `json:"description"`      // 描述
	Status      string     `json:"status"`          // active/resolved
	Source      string     `json:"source"`          // auto/manual 系统自动/手动上报
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy  string     `json:"resolved_by,omitempty"`
}

func (Alarm) TableName() string { return "alarms" }

// OperationLog 操作日志
type OperationLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	TurbineID  uint      `json:"turbine_id"`
	Turbine    *Turbine  `json:"turbine,omitempty" gorm:"foreignKey:TurbineID"`
	Operator   string    `json:"operator"`     // 操作人
	Action     string    `json:"action"`       // 操作类型
	Reason     string    `json:"reason"`       // 操作原因
	PrevStatus string    `json:"prev_status"`  // 操作前状态
	NewStatus  string    `json:"new_status"`   // 操作后状态
	CreatedAt  time.Time `json:"created_at"`
}

func (OperationLog) TableName() string { return "operation_logs" }

// SystemLog 系统日志
type SystemLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Level     string    `json:"level"`   // info/warning/error
	Module    string    `json:"module"`   // 模块
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func (SystemLog) TableName() string { return "system_logs" }

// DailyStats 每日统计
type DailyStats struct {
	ID            uint    `json:"id" gorm:"primaryKey"`
	TurbineID     uint    `json:"turbine_id"`
	Turbine       *Turbine `json:"turbine,omitempty" gorm:"foreignKey:TurbineID"`
	Date          string  `json:"date"`          // YYYY-MM-DD
	TotalPower    float64 `json:"total_power"`   // 日发电量 kWh
	MaxPower      float64 `json:"max_power"`     // 最大功率
	MinPower      float64 `json:"min_power"`     // 最小功率
	AvgPower      float64 `json:"avg_power"`     // 平均功率
	AvgWindSpeed  float64 `json:"avg_wind_speed"`
	Availability float64 `json:"availability"`   // 可用率 %
	FaultCount    int     `json:"fault_count"`    // 故障次数
	RunHours     float64 `json:"run_hours"`      // 运行小时数
}

func (DailyStats) TableName() string { return "daily_stats" }

// User 用户
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"uniqueIndex;size:50"` // 用户名
	Password  string    `json:"-"`                                    // bcrypt 哈希，不返回
	Nickname  string    `json:"nickname" gorm:"size:50"`              // 昵称
	Role      string    `json:"role" gorm:"size:20;default:operator"` // admin / operator
	Status    string    `json:"status" gorm:"size:20;default:active"` // active / disabled
	CreatedAt time.Time `json:"created_at"`
}

func (User) TableName() string { return "users" }

// ─── GORM Hooks ───

func (a *Alarm) AfterCreate(tx *gorm.DB) error {
	// 报警创建后写入系统日志
	level := "info"
	if a.Severity == SeverityWarning {
		level = "warning"
	} else if a.Severity == SeverityCritical {
		level = "error"
	}
	tx.Create(&SystemLog{
		Level:   level,
		Module:  "alarm",
		Message: a.Title + " - " + a.Description,
	})
	return nil
}
