package model

type Script struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"file_path"` // 主jmx文件路径
	FileName    string `json:"file_name,omitempty"`
	FileCount   int64  `json:"file_count,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type ScriptFile struct {
	ID        int64  `json:"id"`
	ScriptID  int64  `json:"script_id"`
	FileName  string `json:"file_name"`
	FilePath  string `json:"file_path"`
	FileType  string `json:"file_type"` // jmx/csv/jar/other
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ScriptVersion struct {
	ID            int64  `json:"id"`
	ScriptID      int64  `json:"script_id"`
	VersionNumber int    `json:"version_number"`
	Content       string `json:"content,omitempty"` // 列表时omit，详情时返回
	ContentHash   string `json:"content_hash,omitempty"`
	ChangeSummary string `json:"change_summary"`
	CreatedAt     string `json:"created_at"`
}

type ScriptDependencyReport struct {
	CSVFiles            []string               `json:"csv_files"`
	FileDependencies    []string               `json:"file_dependencies"`
	PluginDependencies  []string               `json:"plugin_dependencies"`
	AttachedFiles       []string               `json:"attached_files"`
	MissingDependencies []string               `json:"missing_dependencies"`
	Warnings            []string               `json:"warnings"`
	Preflight           *ScriptPreflightReport `json:"preflight,omitempty"`
}

type ScriptStats struct {
	TotalScripts int64 `json:"total_scripts"`
	TotalFiles   int64 `json:"total_files"`
}

type ScriptPreflightFact struct {
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail,omitempty"`
	Tone   string `json:"tone,omitempty"`
}

type ScriptPreflightReport struct {
	Score           int                   `json:"score"`
	Level           string                `json:"level"`
	Summary         string                `json:"summary"`
	MetricMode      string                `json:"metric_mode"`
	MetricUnit      string                `json:"metric_unit"`
	Highlights      []string              `json:"highlights,omitempty"`
	Recommendations []string              `json:"recommendations,omitempty"`
	Facts           []ScriptPreflightFact `json:"facts,omitempty"`
}
