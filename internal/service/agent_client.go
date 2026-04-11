package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

// AgentClient 与 Slave Agent 通信的客户端
type AgentClient struct {
	Host  string
	Port  int
	Token string
}

type AgentCallbackCheckResult struct {
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type AgentEnvironmentReport struct {
	AgentVersion          string   `json:"agent_version"`
	JMeterPath            string   `json:"jmeter_path"`
	JMeterHome            string   `json:"jmeter_home"`
	JMeterVersion         string   `json:"jmeter_version"`
	JMeterVersionRaw      string   `json:"jmeter_version_raw"`
	PluginJars            []string `json:"plugin_jars"`
	PluginFingerprint     string   `json:"plugin_fingerprint"`
	PropertiesLines       []string `json:"properties_lines"`
	PropertiesFingerprint string   `json:"properties_fingerprint"`
	Warnings              []string `json:"warnings,omitempty"`
}

// NewAgentClient 从 Slave 模型创建客户端
func NewAgentClient(host string, port int, token string) *AgentClient {
	return &AgentClient{
		Host:  host,
		Port:  port,
		Token: token,
	}
}

// getBaseURL 获取基础 URL
func (c *AgentClient) getBaseURL() string {
	return fmt.Sprintf("http://%s:%d", c.Host, c.Port)
}

// setAuthHeader 设置认证头
func (c *AgentClient) setAuthHeader(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

// UploadFile 上传文件到 Agent
// localPath: 本地文件路径
// targetName: Agent 上保存的文件名（原始文件名，如 data.csv）
func (c *AgentClient) UploadFile(localPath, targetName string) error {
	// 打开本地文件
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()

	// 创建 multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 添加文件字段
	part, err := writer.CreateFormFile("file", targetName)
	if err != nil {
		return fmt.Errorf("创建form file失败: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("复制文件内容失败: %w", err)
	}

	// 添加 target_name 字段
	if err := writer.WriteField("target_name", targetName); err != nil {
		return fmt.Errorf("写入target_name字段失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭writer失败: %w", err)
	}

	// 创建请求
	url := c.getBaseURL() + "/api/files/upload"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.setAuthHeader(req)

	// 发送请求，超时60秒
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("上传文件请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("上传文件失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteFile 删除 Agent 上的文件
func (c *AgentClient) DeleteFile(filename string) error {
	url := fmt.Sprintf("%s/api/files/%s", c.getBaseURL(), filename)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	c.setAuthHeader(req)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("删除文件请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("删除文件失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteByPrefix 批量删除 Agent 上的文件
func (c *AgentClient) DeleteByPrefix(prefix string) error {
	url := c.getBaseURL() + "/api/files/batch"

	reqBody, err := json.Marshal(map[string]string{"prefix": prefix})
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(req)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("批量删除文件请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("批量删除文件失败，状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Health 检测 Agent 健康状态
func (c *AgentClient) Health() error {
	url := c.getBaseURL() + "/health"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	c.setAuthHeader(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("健康检查请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Agent健康检查失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

func (c *AgentClient) CheckCallback(targetURL string) (*AgentCallbackCheckResult, error) {
	payload, err := json.Marshal(map[string]string{"url": targetURL})
	if err != nil {
		return nil, fmt.Errorf("序列化回调检测请求失败: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.getBaseURL()+"/api/network/check-callback", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("创建回调检测请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuthHeader(req)

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("回调检测请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("回调检测失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result AgentCallbackCheckResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析回调检测响应失败: %w", err)
	}
	return &result, nil
}

func (c *AgentClient) GetEnvironmentReport() (*AgentEnvironmentReport, error) {
	req, err := http.NewRequest(http.MethodGet, c.getBaseURL()+"/api/environment/report", nil)
	if err != nil {
		return nil, fmt.Errorf("创建环境报告请求失败: %w", err)
	}
	c.setAuthHeader(req)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("环境报告请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取环境报告失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var report AgentEnvironmentReport
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, fmt.Errorf("解析环境报告失败: %w", err)
	}
	return &report, nil
}
