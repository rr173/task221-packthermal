// Package model 定义冷链包装热阻反演服务的领域实体与状态。
package model

import "time"

// 试验状态机：计划 → 采集中 → 待反演 → 已确认 → 封存。
const (
	TrialPlanned          = "planned"
	TrialCollecting       = "collecting"
	TrialPendingInversion = "pending_inversion"
	TrialConfirmed        = "confirmed"
	TrialArchived         = "archived"
)

// 温度序列状态：待校验 → 有效 / 接触异常 / 缺口。
const (
	SeriesPending   = "pending"
	SeriesValid     = "valid"
	SeriesContact   = "contact_anomaly"
	SeriesGap       = "gap"
)

// 热模型状态：草稿 → 可运行 / 边界冲突 → 确认。
const (
	ModelDraft   = "draft"
	ModelRunnable = "runnable"
	ModelConflict = "boundary_conflict"
	ModelConfirmed = "confirmed"
)

// 快照状态：计算中 → 待复核 → 发布 → 替代。
const (
	SnapshotComputing = "computing"
	SnapshotReviewing = "under_review"
	SnapshotPublished = "published"
	SnapshotSuperseded = "superseded"
)

// Trial 是一次冷链包装保温验证试验。
type Trial struct {
	ID        int64
	Code      string // 试验编号，唯一
	Title     string
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Layer 是箱体层序中的一层（材料、厚度、初值热物性）。
type Layer struct {
	ID             int64
	TrialID        int64
	Seq            int     // 从外到内的层序号，从 1 开始
	Material       string
	ThicknessMM    float64 // 厚度（毫米）
	Conductivity   float64 // 初始导热系数 W/(m·K)
	Density        float64 // 密度 kg/m³
	SpecificHeat   float64 // 比热 J/(kg·K)
	AreaM2         float64 // 传热面积 m²
}

// EnvProfile 是环境温度阶跃曲线（时间 → 温度）。
type EnvProfile struct {
	ID        int64
	TrialID   int64
	Name      string
	Fingerprint string // 曲线内容指纹，幂等
	Samples   []EnvSample
	CreatedAt time.Time
}

// EnvSample 是环境曲线上的一个采样点。
type EnvSample struct {
	Seconds     float64 // 相对起始时刻（秒）
	Temperature float64 // 温度
}

// Sensor 是布置在箱体上的一只温度传感器。
type Sensor struct {
	ID       int64
	TrialID  int64
	Code     string
	Position string // inner / outer / interface-<n>
	Scale    string // celsius / kelvin
	CreatedAt time.Time
}

// TempSeries 是某只传感器在试验期间上传的温度时序。
type TempSeries struct {
	ID          int64
	TrialID     int64
	SensorID    int64
	Fingerprint string // 内容指纹，幂等
	Scale       string
	State       string
	SampleCount int
	CreatedAt   time.Time
}

// TempSample 是温度时序上的一个采样点。
type TempSample struct {
	Seconds     float64
	Temperature float64
}

// ThermalModel 描述用于反演的热网络参数设定。
type ThermalModel struct {
	ID        int64
	TrialID   int64
	Name      string
	State     string
	Notes     string
	CreatedAt time.Time
}

// InversionResult 是一次热阻/热容反演的拟合结果。
type InversionResult struct {
	ID            int64
	TrialID       int64
	ModelID       int64
	SeriesID      int64  // 用于反演的内壁参考序列
	REff          float64 // 等效热阻 K/W
	CEff          float64 // 等效热容 J/K
	Tau           float64 // 时间常数 s = R*C
	RmsResidual   float64 // 拟合残差均方根 K
	Converged     bool
	Iterations    int
	CreatedAt     time.Time
}

// Snapshot 是发布的一次不可变验证快照。
type Snapshot struct {
	ID        int64
	TrialID   int64
	Version   int
	State     string
	Summary   string
	CreatedAt time.Time
}

// AuditEvent 是一条审计台账。
type AuditEvent struct {
	ID        int64
	TrialID   int64
	Action    string
	Detail    string
	CreatedAt time.Time
}
