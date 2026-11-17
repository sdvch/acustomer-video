// Package model 定义本项目核心数据结构（对齐 SOP 八类对象）。
package model

// 合成任务状态机
const (
	JobQueued     = "queued"
	JobClaimed    = "claimed"
	JobGenerating = "generating"
	JobUploading  = "uploading"
	JobSucceeded  = "succeeded"
	JobFailed     = "failed"
	JobSuperseded = "superseded"
)

// 素材类型
const (
	AssetImage = 1
	AssetVideo = 2
)

// 用户状态
const (
	UserActive  = 1
	UserBlocked = 0
)

// 管理员角色
const (
	AdminRoleSuper    = "super"
	AdminRoleOperator = "operator"
)

// User 账户
type User struct {
	ID           int64  `json:"id"`
	OpenID       string `json:"open_id"`
	Platform     string `json:"platform"`
	Nickname     string `json:"nickname"`
	Phone        string `json:"phone"`
	PasswordHash string `json:"-"`
	Status       int    `json:"status"`
	MemberTier   string `json:"member_tier"`
	MonthQuota   int    `json:"month_quota"`
	UsedQuota    int    `json:"used_quota"`
	CreatedAt    string `json:"created_at"`
}

// Admin 管理员（与用户体系完全隔离）
type Admin struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	Role         string `json:"role"` // super / operator
	Status       int    `json:"status"`
	LastLoginAt  string `json:"last_login_at"`
	CreatedAt    string `json:"created_at"`
}

// Asset 素材（SOP 表14）
type Asset struct {
	ID            int64   `json:"id"`
	UserID        int64   `json:"user_id"`
	ShotNo        int     `json:"shot_no"`
	Type          int     `json:"type"` // 1图 2视频
	ObjectKey     string  `json:"object_key"`
	PreviewURL    string  `json:"preview_url"`
	Mime          string  `json:"mime"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	DurationMs    int64   `json:"duration_ms"`
	SizeBytes     int64   `json:"size_bytes"`
	QcStatus      string  `json:"qc_status"`
	QcJSON        string  `json:"qc_json"`        // 质检警告等结构化信息
	ProbeVerified bool    `json:"probe_verified"` // 是否经过服务端 ffprobe 校验
	TagsJSON      string  `json:"tags_json"`
	UsableStartMs int64   `json:"usable_start_ms"`
	UsableEndMs   int64   `json:"usable_end_ms"`
	SortOrder     int     `json:"sort_order"`
	Authorized    bool    `json:"authorized"`
	DeletedAt     *string `json:"deleted_at"`
	CreatedAt     string  `json:"created_at"`
}

// ComposeProject 合成项目
type ComposeProject struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	Title        string `json:"title"`
	IndustryID   int64  `json:"industry_id"`
	AspectRatio  string `json:"aspect_ratio"`
	TargetWidth  int    `json:"target_width"`
	TargetHeight int    `json:"target_height"`
	TargetFPS    int    `json:"target_fps"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// Video 成片
type Video struct {
	ID                int64  `json:"id"`
	ProjectID         int64  `json:"project_id"`
	JobID             int64  `json:"job_id"`
	ObjectKey         string `json:"object_key"`
	FileURL           string `json:"file_url"`
	PosterURL         string `json:"poster_url"`
	DurationMs        int64  `json:"duration_ms"`
	Width             int    `json:"width"`
	Height            int    `json:"height"`
	FPS               int    `json:"fps"`
	BitrateBps        int64  `json:"bitrate_bps"`
	Codec             string `json:"codec"`
	SubtitleTrackJSON string `json:"subtitle_track_json"` // 预留
	BGMURL            string `json:"bgm_url"`             // 预留
	AcceptStatus      string `json:"accept_status"`
	PublishStatus     string `json:"publish_status"`
	CreatedAt         string `json:"created_at"`
}

// ComposeJob 合成任务
type ComposeJob struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	ProjectID     int64  `json:"project_id"`
	Status        string `json:"status"`
	Progress      int    `json:"progress"` // 0-100
	Stage         string `json:"stage"`
	ClaimToken    string `json:"-"`
	HeartbeatAt   string `json:"-"`
	InputJSON     string `json:"-"`
	ResultVideoID int64  `json:"result_video_id"`
	ErrorMessage  string `json:"error_message"`
	RetryCount    int    `json:"retry_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// JobInput 合成输入快照（写入 compose_job.input_json）
type JobInput struct {
	AssetIDs       []int64 `json:"asset_ids"`
	DurationPolicy string  `json:"duration_policy"` // image_default / fixed
	ClipSeconds    int     `json:"clip_seconds"`
	FitMode        string  `json:"fit_mode"`   // pad / crop
	SegmentMs      []int64 `json:"segment_ms"` // 发起时算好的逐段时长（毫秒），与 asset_ids 一一对应
	TotalMs        int64   `json:"total_ms"`   // 预计成片时长
}
