package model

type Execution struct {
	ID          int64  `json:"id"`
	ScriptID    int64  `json:"script_id"`
	ScriptName  string `json:"script_name"` // 冗余字段方便展示
	SlaveIDs    string `json:"slave_ids"`   // JSON数组
	Status      string `json:"status"`      // running/success/failed/stopped
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"` // 执行结束时间
	Duration    int64  `json:"duration"` // 执行时长(秒)
	Remarks     string `json:"remarks"`  // 执行备注
	ResultPath  string `json:"result_path"`
	ReportPath  string `json:"report_path"`
	SummaryData string `json:"summary_data"` // JSON对象
	LogPath     string `json:"log_path"`
	CreatedAt   string `json:"created_at"`
}
