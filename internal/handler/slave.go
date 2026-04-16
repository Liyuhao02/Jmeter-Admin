package handler

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/model"
	"jmeter-admin/internal/service"

	"github.com/gin-gonic/gin"
)

// ListSlaves GET /api/slaves
func ListSlaves(c *gin.Context) {
	slaves, err := service.ListSlaves()
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}
	c.JSON(http.StatusOK, model.Success(slaves))
}

// CreateSlaveRequest 创建slave请求参数
type CreateSlaveRequest struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	AgentPort  int    `json:"agent_port"`
	AgentToken string `json:"agent_token"`
}

// CreateSlave POST /api/slaves
func CreateSlave(c *gin.Context) {
	var req CreateSlaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Error("参数错误: "+err.Error()))
		return
	}

	// 默认 Agent 端口
	if req.AgentPort == 0 {
		req.AgentPort = 8089
	}

	slave, err := service.CreateSlave(req.Name, req.Host, req.Port, req.AgentPort, req.AgentToken)
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(slave))
}

// UpdateSlaveRequest 更新slave请求参数
type UpdateSlaveRequest struct {
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	AgentPort  int    `json:"agent_port"`
	AgentToken string `json:"agent_token"`
}

// UpdateSlave PUT /api/slaves/:id
func UpdateSlave(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, model.Error("无效的ID"))
		return
	}

	var req UpdateSlaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Error("参数错误: "+err.Error()))
		return
	}

	// 默认 Agent 端口
	if req.AgentPort == 0 {
		req.AgentPort = 8089
	}

	if err := service.UpdateSlave(id, req.Name, req.Host, req.Port, req.AgentPort, req.AgentToken); err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

// DeleteSlave DELETE /api/slaves/:id
func DeleteSlave(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, model.Error("无效的ID"))
		return
	}

	if err := service.DeleteSlave(id); err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

// CheckSlave POST /api/slaves/:id/check
func CheckSlave(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusOK, model.Error("无效的ID"))
		return
	}

	// 同时检测 JMeter RMI 和 Agent
	result, err := service.CheckSlaveBoth(id)
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	// 构建状态字符串
	jmeterStatus := "offline"
	if result.JMeterOnline {
		jmeterStatus = "online"
	}
	agentStatus := "offline"
	if result.AgentOnline {
		agentStatus = "online"
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"online":           result.JMeterOnline,
		"status":           jmeterStatus,
		"agent_online":     result.AgentOnline,
		"agent_status":     agentStatus,
		"last_check_time":  result.LastCheckTime,
		"agent_check_time": result.AgentCheckTime,
		"system_stats":     result.SystemStats,
		"environment_info": result.EnvironmentInfo,
		"agent_uptime":     result.AgentUptime,
		"diagnostic":       result,
	}))
}

func GetSlavePreflight(c *gin.Context) {
	masterHost := strings.TrimSpace(c.Query("master_host"))
	rawIDs := strings.TrimSpace(c.Query("ids"))
	var slaveIDs []int64
	if rawIDs != "" {
		for _, part := range strings.Split(rawIDs, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, model.Error("无效的 Slave ID"))
				return
			}
			slaveIDs = append(slaveIDs, id)
		}
	}

	report, err := service.GetSlavePreflightReport(slaveIDs, masterHost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(report))
}

// GetNetworkInterfaces 获取本机网卡 IP 列表
func GetNetworkInterfaces(c *gin.Context) {
	interfaces, err := net.Interfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error("获取网络接口失败"))
		return
	}

	type NetworkInfo struct {
		Name string `json:"name"`
		IP   string `json:"ip"`
	}

	var result []NetworkInfo
	for _, iface := range interfaces {
		// 跳过回环和未启用的接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			// 只要 IPv4
			if ip == nil || ip.To4() == nil {
				continue
			}
			result = append(result, NetworkInfo{
				Name: iface.Name,
				IP:   ip.String(),
			})
		}
	}

	c.JSON(http.StatusOK, model.Success(result))
}

// GetMasterHostname 获取当前配置的 master hostname
func GetMasterHostname(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{
		"master_hostname": config.GlobalConfig.JMeter.MasterHostname,
	}))
}

// UpdateMasterHostname 更新 master hostname
func UpdateMasterHostname(c *gin.Context) {
	var req struct {
		MasterHostname string `json:"master_hostname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的请求参数"))
		return
	}

	// 更新内存中的配置
	config.GlobalConfig.JMeter.MasterHostname = req.MasterHostname

	// 持久化到 config.yaml
	if err := config.SaveConfig(""); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error("保存配置失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"master_hostname": req.MasterHostname,
	}))
}

// GetHeartbeatStatus 获取所有 Slave 的心跳状态
func GetHeartbeatStatus(c *gin.Context) {
	slaves, err := service.ListSlaves()
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	// 构建心跳状态列表
	type HeartbeatInfo struct {
		ID              int64  `json:"id"`
		Name            string `json:"name"`
		Host            string `json:"host"`
		Port            int    `json:"port"`
		Status          string `json:"status"`
		AgentPort       int    `json:"agent_port"`
		AgentStatus     string `json:"agent_status"`
		LastCheckTime   string `json:"last_check_time"`
		AgentCheckTime  string `json:"agent_check_time"`
		SystemStats     string `json:"system_stats"`
		EnvironmentInfo string `json:"environment_info"`
		AgentUptime     int64  `json:"agent_uptime"`
	}

	var heartbeatList []HeartbeatInfo
	for _, slave := range slaves {
		heartbeatList = append(heartbeatList, HeartbeatInfo{
			ID:              slave.ID,
			Name:            slave.Name,
			Host:            slave.Host,
			Port:            slave.Port,
			Status:          slave.Status,
			AgentPort:       slave.AgentPort,
			AgentStatus:     slave.AgentStatus,
			LastCheckTime:   slave.LastCheckTime,
			AgentCheckTime:  slave.AgentCheckTime,
			SystemStats:     slave.SystemStats,
			EnvironmentInfo: slave.EnvironmentInfo,
			AgentUptime:     slave.AgentUptime,
		})
	}

	masterHost := config.GlobalConfig.JMeter.MasterHostname
	if masterHost == "" {
		masterHost = "localhost"
	}

	masterEnvironment := service.GetLocalEnvironmentReport()

	c.JSON(http.StatusOK, model.Success(gin.H{
		"master": gin.H{
			"id":               0,
			"name":             "Master",
			"host":             masterHost,
			"status":           "online",
			"agent_status":     "online",
			"system_stats":     collectMasterSystemStats(),
			"environment_info": masterEnvironment,
			"agent_uptime":     0,
			"last_check_time":  time.Now().Format("2006-01-02 15:04:05"),
		},
		"slaves":          heartbeatList,
		"check_interval":  30,
		"last_check_time": time.Now().Format("2006-01-02 15:04:05"),
	}))
}
