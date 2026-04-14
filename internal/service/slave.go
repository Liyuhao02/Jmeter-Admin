package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
)

// ListSlaves 查询所有slave列表
func ListSlaves() ([]model.Slave, error) {
	rows, err := database.DB.Query("SELECT id, name, host, port, status, agent_status, agent_port, agent_token, last_check_time, agent_check_time, system_stats, agent_uptime, created_at FROM slaves ORDER BY id DESC")
	if err != nil {
		return nil, fmt.Errorf("查询slave列表失败: %w", err)
	}
	defer rows.Close()

	var slaves []model.Slave
	for rows.Next() {
		var slave model.Slave
		var lastCheckTime sql.NullString
		var agentCheckTime sql.NullString
		var systemStats sql.NullString
		var agentStatus sql.NullString
		if err := rows.Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &agentStatus, &slave.AgentPort, &slave.AgentToken, &lastCheckTime, &agentCheckTime, &systemStats, &slave.AgentUptime, &slave.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描slave数据失败: %w", err)
		}
		if lastCheckTime.Valid {
			slave.LastCheckTime = lastCheckTime.String
		}
		if agentCheckTime.Valid {
			slave.AgentCheckTime = agentCheckTime.String
		}
		if systemStats.Valid {
			slave.SystemStats = systemStats.String
		}
		if agentStatus.Valid {
			slave.AgentStatus = agentStatus.String
		} else {
			slave.AgentStatus = "unknown"
		}
		slaves = append(slaves, slave)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历slave数据失败: %w", err)
	}

	return slaves, nil
}

// CreateSlave 创建slave
func CreateSlave(name, host string, port int, agentPort int, agentToken string) (*model.Slave, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := database.DB.Exec(
		"INSERT INTO slaves (name, host, port, status, agent_port, agent_token, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		name, host, port, "offline", agentPort, agentToken, now,
	)
	if err != nil {
		return nil, fmt.Errorf("创建slave失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取slave ID失败: %w", err)
	}

	slave := &model.Slave{
		ID:         id,
		Name:       name,
		Host:       host,
		Port:       port,
		Status:     "offline",
		AgentPort:  agentPort,
		AgentToken: agentToken,
		CreatedAt:  now,
	}

	return slave, nil
}

// UpdateSlave 更新slave
func UpdateSlave(id int64, name, host string, port int, agentPort int, agentToken string) error {
	result, err := database.DB.Exec(
		"UPDATE slaves SET name = ?, host = ?, port = ?, agent_port = ?, agent_token = ? WHERE id = ?",
		name, host, port, agentPort, agentToken, id,
	)
	if err != nil {
		return fmt.Errorf("更新slave失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slave不存在")
	}

	return nil
}

// DeleteSlave 删除slave
func DeleteSlave(id int64) error {
	result, err := database.DB.Exec("DELETE FROM slaves WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除slave失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slave不存在")
	}

	return nil
}

// CheckSlave 检测slave连通性（JMeter RMI 端口）
func CheckSlave(id int64) (bool, error) {
	// 获取slave信息
	var slave model.Slave
	var lastCheckTime sql.NullString
	var agentCheckTime sql.NullString
	var systemStats sql.NullString
	var agentStatus sql.NullString
	err := database.DB.QueryRow(
		"SELECT id, name, host, port, status, agent_status, agent_port, agent_token, last_check_time, agent_check_time, system_stats, agent_uptime, created_at FROM slaves WHERE id = ?",
		id,
	).Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &agentStatus, &slave.AgentPort, &slave.AgentToken, &lastCheckTime, &agentCheckTime, &systemStats, &slave.AgentUptime, &slave.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("slave不存在")
		}
		return false, fmt.Errorf("查询slave失败: %w", err)
	}
	if lastCheckTime.Valid {
		slave.LastCheckTime = lastCheckTime.String
	}
	if agentCheckTime.Valid {
		slave.AgentCheckTime = agentCheckTime.String
	}
	if systemStats.Valid {
		slave.SystemStats = systemStats.String
	}
	if agentStatus.Valid {
		slave.AgentStatus = agentStatus.String
	} else {
		slave.AgentStatus = "unknown"
	}

	// 检测 JMeter RMI 连通性
	address := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	var isOnline bool
	if err != nil {
		isOnline = false
	} else {
		isOnline = true
		conn.Close()
	}

	// 更新状态和最后检测时间
	status := "offline"
	if isOnline {
		status = "online"
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err = database.DB.Exec("UPDATE slaves SET status = ?, last_check_time = ? WHERE id = ?", status, now, id)
	if err != nil {
		return isOnline, fmt.Errorf("更新slave状态失败: %w", err)
	}

	return isOnline, nil
}

// DiagnosticResult 诊断结果结构体
type DiagnosticResult struct {
	JMeterOnline  bool   `json:"jmeter_online"`
	AgentOnline   bool   `json:"agent_online"`
	JMeterError   string `json:"jmeter_error,omitempty"` // "connection_refused" / "timeout" / "unknown"
	AgentError    string `json:"agent_error,omitempty"`  // "connection_refused" / "timeout" / "auth_failed" / "unknown"
	JMeterLatency int64  `json:"jmeter_latency_ms"`
	AgentLatency  int64  `json:"agent_latency_ms"`
	Suggestion    string `json:"suggestion,omitempty"`
}

type CallbackReachabilityResult struct {
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

// classifyNetworkError 分类网络错误类型
func classifyNetworkError(err error) string {
	if err == nil {
		return ""
	}

	// 检查是否为超时错误
	if os.IsTimeout(err) {
		return "timeout"
	}

	// 检查 net.Error 接口
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
	}

	// 检查 net.OpError
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Timeout() {
			return "timeout"
		}
		// 检查系统调用错误
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
			if errors.Is(sysErr.Err, syscall.ECONNREFUSED) {
				return "connection_refused"
			}
		}
		// 直接检查错误字符串
		if strings.Contains(opErr.Err.Error(), "connection refused") {
			return "connection_refused"
		}
	}

	// 检查错误字符串
	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "ECONNREFUSED") ||
		strings.Contains(errStr, "Connection refused") {
		return "connection_refused"
	}
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "Timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "context deadline") {
		return "timeout"
	}

	return "unknown"
}

func CheckSlaveCallbackReachability(slave model.Slave, callbackURL string) (*CallbackReachabilityResult, error) {
	client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
	result, err := client.CheckCallback(callbackURL)
	if err != nil {
		return nil, err
	}
	return &CallbackReachabilityResult{
		Reachable:  result.Reachable,
		StatusCode: result.StatusCode,
		LatencyMS:  result.LatencyMS,
		Error:      result.Error,
	}, nil
}

func parseAgentSystemStats(raw string) *model.AgentSystemStats {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var stats model.AgentSystemStats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		return nil
	}
	return &stats
}

func GetSlavePreflightReport(slaveIDs []int64, masterHost string) (*model.SlavePreflightReport, error) {
	slaves, err := ListSlaves()
	if err != nil {
		return nil, err
	}

	selectedSet := make(map[int64]bool)
	for _, id := range slaveIDs {
		selectedSet[id] = true
	}

	selected := make([]model.Slave, 0, len(slaveIDs))
	for _, slave := range slaves {
		if len(selectedSet) > 0 && !selectedSet[slave.ID] {
			continue
		}
		selected = append(selected, slave)
	}

	callbackURL := ""
	trimmedMasterHost := strings.TrimSpace(masterHost)
	if trimmedMasterHost == "" {
		trimmedMasterHost = strings.TrimSpace(config.GlobalConfig.JMeter.MasterHostname)
	}
	if trimmedMasterHost != "" {
		callbackURL = fmt.Sprintf("http://%s:%d/api/executions/callback-probe", trimmedMasterHost, config.GlobalConfig.Server.Port)
	}

	report := &model.SlavePreflightReport{
		MasterHost:        trimmedMasterHost,
		MasterCallbackURL: callbackURL,
		SelectedCount:     len(selected),
		Warnings:          []string{},
		Nodes:             make([]model.SlavePreflightNode, 0, len(selected)),
	}

	if trimmedMasterHost == "" {
		report.Warnings = append(report.Warnings, "未配置 Master 回调地址，当前只能检测节点存活，无法验证 Slave 是否能把结果回传回来。")
	}

	for _, slave := range selected {
		diag, err := CheckSlaveBoth(slave.ID)
		if err != nil {
			report.Nodes = append(report.Nodes, model.SlavePreflightNode{
				ID:         slave.ID,
				Name:       slave.Name,
				Host:       slave.Host,
				Port:       slave.Port,
				AgentPort:  slave.AgentPort,
				Suggestion: err.Error(),
			})
			continue
		}

		node := model.SlavePreflightNode{
			ID:           slave.ID,
			Name:         slave.Name,
			Host:         slave.Host,
			Port:         slave.Port,
			AgentPort:    slave.AgentPort,
			JMeterOnline: diag.JMeterOnline,
			AgentOnline:  diag.AgentOnline,
			Suggestion:   diag.Suggestion,
			SystemStats:  parseAgentSystemStats(slave.SystemStats),
		}

		if callbackURL != "" && diag.AgentOnline {
			result, callbackErr := CheckSlaveCallbackReachability(slave, callbackURL)
			if callbackErr != nil {
				node.CallbackError = callbackErr.Error()
			} else if result != nil {
				node.CallbackReachable = result.Reachable
				node.CallbackLatencyMS = result.LatencyMS
				node.CallbackStatusCode = result.StatusCode
				node.CallbackError = result.Error
			}
		}

		if node.JMeterOnline && node.AgentOnline && (callbackURL == "" || node.CallbackReachable) {
			report.ReadyCount += 1
		}

		report.Nodes = append(report.Nodes, node)
	}

	if report.SelectedCount == 0 {
		report.Warnings = append(report.Warnings, "当前未选中任何 Slave 节点。")
	}
	if report.ReadyCount < report.SelectedCount && report.SelectedCount > 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("仅 %d/%d 个节点通过执行前体检，建议优先修复异常节点后再执行。", report.ReadyCount, report.SelectedCount))
	}

	return report, nil
}

// generateSuggestion 根据诊断结果生成建议
func generateSuggestion(result *DiagnosticResult, slave *model.Slave) {
	var suggestions []string

	if !result.JMeterOnline {
		switch result.JMeterError {
		case "connection_refused":
			suggestions = append(suggestions, fmt.Sprintf("JMeter RMI 端口 %d 未监听，请检查 jmeter-server 是否已启动", slave.Port))
		case "timeout":
			suggestions = append(suggestions, fmt.Sprintf("JMeter RMI 连接超时，请检查防火墙是否放行 %d 端口", slave.Port))
		default:
			suggestions = append(suggestions, fmt.Sprintf("JMeter RMI 连接异常，请检查 %s:%d 是否可达", slave.Host, slave.Port))
		}
	}

	if !result.AgentOnline {
		switch result.AgentError {
		case "connection_refused":
			suggestions = append(suggestions, fmt.Sprintf("Agent 服务未运行，请启动 jmeter-agent (端口 %d)", slave.AgentPort))
		case "timeout":
			suggestions = append(suggestions, fmt.Sprintf("Agent 连接超时，请检查防火墙是否放行 %d 端口", slave.AgentPort))
		case "auth_failed":
			suggestions = append(suggestions, "Agent 认证失败，请检查 Token 配置是否正确")
		default:
			suggestions = append(suggestions, fmt.Sprintf("Agent 连接异常，请检查 %s:%d 是否可达", slave.Host, slave.AgentPort))
		}
	}

	if len(suggestions) == 0 {
		result.Suggestion = ""
	} else if len(suggestions) == 1 {
		result.Suggestion = suggestions[0]
	} else {
		result.Suggestion = strings.Join(suggestions, "；")
	}
}

// AgentHealthResult Agent 健康检测结果
type AgentHealthResult struct {
	Online      bool
	ErrorType   string
	Latency     int64
	SystemStats string // JSON string
	AgentUptime int64
}

// CheckSlaveAgent 检测 Slave Agent HTTP 服务连通性
// 返回: AgentHealthResult, 错误
func CheckSlaveAgent(slave *model.Slave) (*AgentHealthResult, error) {
	result := &AgentHealthResult{
		Online:      false,
		ErrorType:   "",
		Latency:     0,
		SystemStats: "",
		AgentUptime: 0,
	}

	if slave.AgentPort == 0 {
		slave.AgentPort = 8089 // 默认端口
	}

	url := fmt.Sprintf("http://%s:%d/health", slave.Host, slave.AgentPort)
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		result.ErrorType = "unknown"
		return result, err
	}

	// 如果配置了 token，添加到请求头
	if slave.AgentToken != "" {
		req.Header.Set("Authorization", "Bearer "+slave.AgentToken)
	}

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start).Milliseconds()

	if err != nil {
		result.ErrorType = classifyNetworkError(err)
		return result, err
	}
	defer resp.Body.Close()

	// 检查认证失败
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		result.ErrorType = "auth_failed"
		return result, fmt.Errorf("agent authentication failed: status %d", resp.StatusCode)
	}

	result.Online = resp.StatusCode == http.StatusOK

	// 解析响应体中的 sys_stats 和 uptime_seconds
	if result.Online {
		var healthResp struct {
			Status        string          `json:"status"`
			Version       string          `json:"version"`
			UptimeSeconds int64           `json:"uptime_seconds"`
			SysStats      json.RawMessage `json:"sys_stats"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&healthResp); err == nil {
			// 将 sys_stats 原样存储为 JSON 字符串
			if len(healthResp.SysStats) > 0 {
				result.SystemStats = string(healthResp.SysStats)
			}
			result.AgentUptime = healthResp.UptimeSeconds
		}
	} else {
		// 读取并丢弃响应体
		_, _ = io.Copy(io.Discard, resp.Body)
	}

	return result, nil
}

// CheckSlaveBoth 同时检测 JMeter RMI 和 Agent 状态
// 返回: 诊断结果, error
func CheckSlaveBoth(id int64) (DiagnosticResult, error) {
	result := DiagnosticResult{
		JMeterOnline:  false,
		AgentOnline:   false,
		JMeterLatency: 0,
		AgentLatency:  0,
	}

	// 获取slave信息
	var slave model.Slave
	var lastCheckTime sql.NullString
	var agentCheckTime sql.NullString
	var systemStats sql.NullString
	var agentStatus sql.NullString
	err := database.DB.QueryRow(
		"SELECT id, name, host, port, status, agent_status, agent_port, agent_token, last_check_time, agent_check_time, system_stats, agent_uptime, created_at FROM slaves WHERE id = ?",
		id,
	).Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &agentStatus, &slave.AgentPort, &slave.AgentToken, &lastCheckTime, &agentCheckTime, &systemStats, &slave.AgentUptime, &slave.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return result, fmt.Errorf("slave不存在")
		}
		return result, fmt.Errorf("查询slave失败: %w", err)
	}
	if lastCheckTime.Valid {
		slave.LastCheckTime = lastCheckTime.String
	}
	if agentCheckTime.Valid {
		slave.AgentCheckTime = agentCheckTime.String
	}
	if systemStats.Valid {
		slave.SystemStats = systemStats.String
	}
	if agentStatus.Valid {
		slave.AgentStatus = agentStatus.String
	} else {
		slave.AgentStatus = "unknown"
	}

	// 检测 JMeter RMI 连通性
	jmeterAddress := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
	start := time.Now()
	jmeterConn, jmeterErr := net.DialTimeout("tcp", jmeterAddress, 3*time.Second)
	result.JMeterLatency = time.Since(start).Milliseconds()

	result.JMeterOnline = jmeterErr == nil
	if result.JMeterOnline {
		jmeterConn.Close()
	} else {
		result.JMeterError = classifyNetworkError(jmeterErr)
	}

	// 检测 Agent HTTP 连通性
	agentResult, _ := CheckSlaveAgent(&slave)
	result.AgentOnline = agentResult.Online
	result.AgentError = agentResult.ErrorType
	result.AgentLatency = agentResult.Latency

	// 生成诊断建议
	generateSuggestion(&result, &slave)

	// 更新状态和最后检测时间
	now := time.Now().Format("2006-01-02 15:04:05")
	jmeterStatus := "offline"
	if result.JMeterOnline {
		jmeterStatus = "online"
	}
	newAgentStatus := "offline"
	if result.AgentOnline {
		newAgentStatus = "online"
	}

	_, err = database.DB.Exec(
		"UPDATE slaves SET status = ?, agent_status = ?, last_check_time = ?, agent_check_time = ?, system_stats = ?, agent_uptime = ? WHERE id = ?",
		jmeterStatus, newAgentStatus, now, now, agentResult.SystemStats, agentResult.AgentUptime, id,
	)
	if err != nil {
		return result, fmt.Errorf("更新slave状态失败: %w", err)
	}

	return result, nil
}

// StartHeartbeat 启动定时心跳检测
func StartHeartbeat(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		// 启动时立即执行一次
		checkAllSlaves()
		for range ticker.C {
			checkAllSlaves()
		}
	}()
}

func checkAllSlaves() {
	slaves, err := ListSlaves()
	if err != nil {
		log.Printf("心跳检测: 获取 Slave 列表失败: %v", err)
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数为10

	for _, slave := range slaves {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(s model.Slave) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			// 检测 JMeter RMI 连通性
			address := fmt.Sprintf("%s:%d", s.Host, s.Port)
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)

			var jmeterOnline bool
			if err != nil {
				jmeterOnline = false
			} else {
				jmeterOnline = true
				conn.Close()
			}

			// 检测 Agent HTTP 连通性
			agentResult, _ := CheckSlaveAgent(&s)
			agentOnline := agentResult.Online

			// 更新状态和最后检测时间
			jmeterStatus := "offline"
			if jmeterOnline {
				jmeterStatus = "online"
			}
			agentStatus := "offline"
			if agentOnline {
				agentStatus = "online"
			}
			now := time.Now().Format("2006-01-02 15:04:05")

			_, dbErr := database.DB.Exec(
				"UPDATE slaves SET status = ?, agent_status = ?, last_check_time = ?, agent_check_time = ?, system_stats = ?, agent_uptime = ? WHERE id = ?",
				jmeterStatus, agentStatus, now, now, agentResult.SystemStats, agentResult.AgentUptime, s.ID,
			)
			if dbErr != nil {
				log.Printf("心跳检测: 更新 Slave %s 状态失败: %v", s.Name, dbErr)
			}

			log.Printf("心跳检测: %s (%s:%d) JMeter: %s, Agent: %s",
				s.Name, s.Host, s.Port,
				map[bool]string{true: "online", false: "offline"}[jmeterOnline],
				map[bool]string{true: "online", false: "offline"}[agentOnline],
			)
		}(slave)
	}

	wg.Wait()
}
