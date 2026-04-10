package model

// Agent 系统资源统计
type AgentSystemStats struct {
	CPU     AgentCPUStats     `json:"cpu"`
	Memory  AgentMemoryStats  `json:"memory"`
	Disk    AgentDiskStats    `json:"disk"`
	Network AgentNetworkStats `json:"network"`
}

type AgentCPUStats struct {
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

type AgentMemoryStats struct {
	TotalMB uint64  `json:"total_mb"`
	UsedMB  uint64  `json:"used_mb"`
	Percent float64 `json:"percent"`
}

type AgentDiskStats struct {
	TotalMB uint64  `json:"total_mb"`
	UsedMB  uint64  `json:"used_mb"`
	Percent float64 `json:"percent"`
}

type AgentNetworkStats struct {
	Connections int `json:"connections"`
}

type Slave struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Status         string `json:"status"`       // online/offline
	AgentPort      int    `json:"agent_port"`   // Agent 服务端口，默认 8089
	AgentToken     string `json:"agent_token"`  // Agent 鉴权 token
	AgentStatus    string `json:"agent_status"` // Agent 状态: online/offline/unknown（非数据库字段，运行时填充）
	LastCheckTime  string `json:"last_check_time"`
	AgentCheckTime string `json:"agent_check_time"`       // Agent 最后检测时间
	SystemStats    string `json:"system_stats,omitempty"` // JSON 格式存储
	AgentUptime    int64  `json:"agent_uptime,omitempty"` // Agent 运行秒数
	CreatedAt      string `json:"created_at"`
}
