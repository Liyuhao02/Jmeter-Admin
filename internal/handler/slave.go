package handler

import (
	"net"
	"net/http"
	"strconv"
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
	Name string `json:"name" binding:"required"`
	Host string `json:"host" binding:"required"`
	Port int    `json:"port" binding:"required"`
}

// CreateSlave POST /api/slaves
func CreateSlave(c *gin.Context) {
	var req CreateSlaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, model.Error("参数错误: "+err.Error()))
		return
	}

	slave, err := service.CreateSlave(req.Name, req.Host, req.Port)
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(slave))
}

// UpdateSlaveRequest 更新slave请求参数
type UpdateSlaveRequest struct {
	Name string `json:"name" binding:"required"`
	Host string `json:"host" binding:"required"`
	Port int    `json:"port" binding:"required"`
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

	if err := service.UpdateSlave(id, req.Name, req.Host, req.Port); err != nil {
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

	isOnline, err := service.CheckSlave(id)
	if err != nil {
		c.JSON(http.StatusOK, model.Error(err.Error()))
		return
	}

	// 同时返回 online 和 status 字段，兼容前端
	status := "offline"
	if isOnline {
		status = "online"
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"online": isOnline,
		"status": status,
	}))
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
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Host          string `json:"host"`
		Port          int    `json:"port"`
		Status        string `json:"status"`
		LastCheckTime string `json:"last_check_time"`
	}

	var heartbeatList []HeartbeatInfo
	for _, slave := range slaves {
		heartbeatList = append(heartbeatList, HeartbeatInfo{
			ID:            slave.ID,
			Name:          slave.Name,
			Host:          slave.Host,
			Port:          slave.Port,
			Status:        slave.Status,
			LastCheckTime: slave.LastCheckTime,
		})
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"slaves":          heartbeatList,
		"check_interval":  30,
		"last_check_time": time.Now().Format("2006-01-02 15:04:05"),
	}))
}
