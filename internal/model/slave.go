package model

type Slave struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Status        string `json:"status"` // online/offline
	LastCheckTime string `json:"last_check_time"`
	CreatedAt     string `json:"created_at"`
}
