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
