package model

type Execution struct {
	ID            int64                 `json:"id"`
	ScriptID      int64                 `json:"script_id"`
	ScriptName    string                `json:"script_name"`              // 冗余字段方便展示
	SlaveIDs      string                `json:"slave_ids"`                // JSON数组
	Status        string                `json:"status"`                   // running/success/failed/stopped
	ProcessStatus string                `json:"process_status,omitempty"` // 进程执行状态
	DisplayStatus string                `json:"display_status,omitempty"` // 展示状态：如 completed_with_errors
	ResultStatus  string                `json:"result_status,omitempty"`  // 请求结果状态：healthy/partial_fail/all_fail
	StatusTone    string                `json:"status_tone,omitempty"`    // success/warning/danger/info
	StatusReason  string                `json:"status_reason,omitempty"`  // 更具体的状态说明
	StartTime     string                `json:"start_time"`
	EndTime       string                `json:"end_time"` // 执行结束时间
	Duration      int64                 `json:"duration"` // 执行时长(秒)
	Remarks       string                `json:"remarks"`  // 执行备注
	ResultPath    string                `json:"result_path"`
	ReportPath    string                `json:"report_path"`
	SummaryData   string                `json:"summary_data"` // JSON对象
	LogPath       string                `json:"log_path"`
	IsBaseline    bool                  `json:"is_baseline"` // 是否为基准线
	CreatedAt     string                `json:"created_at"`
	Diagnostics   *ExecutionDiagnostics `json:"diagnostics,omitempty"`
}

type ExecutionFileStatus struct {
	Label  string `json:"label"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	Size   int64  `json:"size"`
}

type ExecutionDiagnostics struct {
	Mode                  string                `json:"mode"`
	IncludeMaster         bool                  `json:"include_master"`
	SlaveCount            int                   `json:"slave_count"`
	SlaveHosts            []string              `json:"slave_hosts,omitempty"`
	ResultFiles           []ExecutionFileStatus `json:"result_files,omitempty"`
	RuntimeScripts        []ExecutionFileStatus `json:"runtime_scripts,omitempty"`
	ResultMergeReady      bool                  `json:"result_merge_ready"`
	SaveHTTPDetails       bool                  `json:"save_http_details"`
	DetailState           string                `json:"detail_state,omitempty"`
	DetailLocalFile       *ExecutionFileStatus  `json:"detail_local_file,omitempty"`
	DetailRemoteFiles     []ExecutionFileStatus `json:"detail_remote_files,omitempty"`
	ExpectedDetailSources []string              `json:"expected_detail_sources,omitempty"`
	ReceivedDetailSources []string              `json:"received_detail_sources,omitempty"`
	MissingDetailSources  []string              `json:"missing_detail_sources,omitempty"`
	SplitCSV              bool                  `json:"split_csv"`
	CSVDependencies       []string              `json:"csv_dependencies,omitempty"`
	FileDependencies      []string              `json:"file_dependencies,omitempty"`
	PluginDependencies    []string              `json:"plugin_dependencies,omitempty"`
	MissingDependencies   []string              `json:"missing_dependencies,omitempty"`
	Warnings              []string              `json:"warnings,omitempty"`
}
