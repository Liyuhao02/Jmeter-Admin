package handler

import (
	"archive/zip"
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
	"jmeter-admin/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	gopsnet "github.com/shirou/gopsutil/v4/net"
)

type UploadExecutionErrorDetailsRequest struct {
	Token   string `json:"token" binding:"required"`
	Source  string `json:"source" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// CreateExecutionRequest 创建执行请求
type CreateExecutionRequest struct {
	ScriptID                  int64   `json:"script_id" binding:"required"`
	SlaveIDs                  []int64 `json:"slave_ids"`
	Remarks                   string  `json:"remarks"`
	SaveHTTPDetails           bool    `json:"save_http_details"`
	IncludeMaster             bool    `json:"include_master"`
	SplitCSV                  bool    `json:"split_csv"` // 是否拆分 CSV 文件
	IgnoreEnvironmentWarnings bool    `json:"ignore_environment_warnings"`
}

// CreateExecution 创建并启动执行
func CreateExecution(c *gin.Context) {
	var req CreateExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的请求参数: "+err.Error()))
		return
	}

	execution, err := service.CreateExecution(req.ScriptID, req.SlaveIDs, req.Remarks, req.SaveHTTPDetails, req.IncludeMaster, req.SplitCSV, req.IgnoreEnvironmentWarnings)
	if err != nil {
		var envErr *service.ExecutionEnvironmentValidationError
		if errors.As(err, &envErr) {
			c.JSON(http.StatusConflict, model.Response{
				Code:    40901,
				Message: envErr.Error(),
				Data: gin.H{
					"can_ignore":         true,
					"environment_report": envErr.Snapshot,
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(execution))
}

// ListExecutions 分页查询执行记录
func ListExecutions(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 获取筛选参数
	scriptID := c.Query("script_id")
	status := c.Query("status")
	keyword := c.Query("keyword")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	executions, total, err := service.ListExecutionsFiltered(page, pageSize, scriptID, status, keyword, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PageSuccess(total, executions))
}

// GetExecutionStats 获取执行统计数据
func GetExecutionStats(c *gin.Context) {
	stats, err := service.GetExecutionStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(stats))
}

// GetExecution 获取执行详情
func GetExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	execution, err := service.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(execution))
}

// GetExecutionLiveMetrics 获取执行中的实时指标与趋势
func GetExecutionLiveMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	metrics, err := service.GetExecutionLiveMetrics(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(metrics))
}

func GetExecutionStream(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID"))
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.Error("当前环境不支持流式输出"))
		return
	}

	var (
		cachedNodes   = []gin.H{}
		lastNodeFetch time.Time
	)

	sendSnapshot := func(forceNodeRefresh bool) bool {
		execution, execErr := service.GetExecution(id)
		if execErr != nil {
			c.SSEvent("error", gin.H{"message": execErr.Error()})
			flusher.Flush()
			return false
		}

		liveMetrics, metricsErr := service.GetExecutionLiveMetrics(id)
		if metricsErr != nil {
			liveMetrics = &service.LiveExecutionMetrics{Status: execution.Status, Points: []service.LiveMetricPoint{}}
		}

		errorOverview, overviewErr := service.GetExecutionErrorOverview(id)
		if overviewErr != nil {
			errorOverview = nil
		}

		if forceNodeRefresh || lastNodeFetch.IsZero() || time.Since(lastNodeFetch) >= 3*time.Second {
			nodes, nodeErr := collectExecutionNodeMetricsData(execution)
			if nodeErr == nil {
				cachedNodes = nodes
				lastNodeFetch = time.Now()
			}
		}

		c.SSEvent("snapshot", gin.H{
			"server_time":    time.Now().Format("2006-01-02 15:04:05"),
			"execution":      execution,
			"live_metrics":   liveMetrics,
			"error_overview": errorOverview,
			"node_metrics":   cachedNodes,
		})
		flusher.Flush()

		if execution.Status != "running" {
			c.SSEvent("complete", gin.H{
				"status": execution.Status,
			})
			flusher.Flush()
			return false
		}
		return true
	}

	if !sendSnapshot(true) {
		return
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			if !sendSnapshot(false) {
				return
			}
		}
	}
}

// StopExecution 停止执行
func StopExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	if err := service.StopExecution(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success("执行已停止"))
}

// DeleteExecution 删除执行记录
func DeleteExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	if err := service.DeleteExecution(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success("执行记录已删除"))
}

// GetExecutionErrors 获取执行错误记录
func GetExecutionErrors(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID"))
		return
	}

	errors, err := service.GetExecutionErrors(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(errors))
}

// UploadExecutionErrorDetails 接收 JMeter 监听器回传的错误明细文件
func UploadExecutionErrorDetails(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID"))
		return
	}

	var req UploadExecutionErrorDetailsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的上传参数"))
		return
	}

	if err := service.SaveUploadedExecutionErrorDetails(id, req.Token, req.Source, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"saved": true,
	}))
}

// CallbackProbe 供 Slave Agent 反向探测 Master 回调可达性
func CallbackProbe(c *gin.Context) {
	c.JSON(http.StatusOK, model.Success(gin.H{
		"reachable":   true,
		"server_time": time.Now().Format("2006-01-02 15:04:05"),
	}))
}

// DownloadResultFile 下载 JTL 结果文件
func DownloadResultFile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	// 获取执行记录
	execution, err := service.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	// 检查文件是否存在
	if execution.ResultPath == "" {
		c.JSON(http.StatusNotFound, model.Error("结果文件路径为空"))
		return
	}

	fileInfo, err := os.Stat(execution.ResultPath)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, model.Error("结果文件不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, model.Error("无法访问结果文件"))
		}
		return
	}

	// 打开文件
	file, err := os.Open(execution.ResultPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error("无法打开结果文件"))
		return
	}
	defer file.Close()

	// 设置响应头
	filename := fmt.Sprintf("execution_%d_result.jtl", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 流式传输文件
	io.Copy(c.Writer, file)
}

// DownloadReport 下载 HTML 报告（打包为 ZIP）
func DownloadReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	// 获取执行记录
	execution, err := service.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	// 检查报告目录是否存在
	if execution.ReportPath == "" {
		c.JSON(http.StatusNotFound, model.Error("报告路径为空"))
		return
	}

	reportDir := execution.ReportPath
	info, err := os.Stat(reportDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, model.Error("报告目录不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, model.Error("无法访问报告目录"))
		}
		return
	}
	if !info.IsDir() {
		c.JSON(http.StatusNotFound, model.Error("报告路径不是目录"))
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("execution_%d_report.zip", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/zip")

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// 遍历报告目录并添加到 ZIP
	err = filepath.Walk(reportDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(reportDir, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建 ZIP 文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if info.IsDir() {
			header.Name += "/"
			_, err = zipWriter.CreateHeader(header)
			return err
		}

		// 写入文件内容
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		// 由于已经开始写入响应，无法返回 JSON 错误
		// 记录错误日志
		fmt.Printf("创建报告 ZIP 失败: %v\n", err)
	}
}

// ExportErrors 导出错误记录为 CSV
func ExportErrors(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	// 获取错误分析数据
	errorAnalysis, err := service.GetExecutionErrors(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	// 检查是否有错误记录
	if errorAnalysis == nil || len(errorAnalysis.Records) == 0 {
		c.JSON(http.StatusNotFound, model.Error("没有错误记录"))
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("execution_%d_errors.csv", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")

	// 写入 UTF-8 BOM
	c.Writer.Write([]byte("\xEF\xBB\xBF"))

	// 创建 CSV 写入器
	csvWriter := csv.NewWriter(c.Writer)
	defer csvWriter.Flush()

	// 写入表头
	csvWriter.Write([]string{"时间戳", "标签", "响应码", "响应信息", "失败原因", "请求URL", "请求头", "请求体", "响应内容", "响应头", "线程名", "响应时间(ms)", "延迟(ms)", "连接时间(ms)", "发送字节数", "接收字节数"})

	// 写入错误记录
	for _, r := range errorAnalysis.Records {
		csvWriter.Write([]string{
			r.Timestamp,
			r.Label,
			r.ResponseCode,
			r.ResponseMessage,
			r.FailureMessage,
			r.URL,
			r.RequestHeaders,
			r.RequestBody,
			r.ResponseData,
			r.ResponseHeaders,
			r.ThreadName,
			fmt.Sprintf("%d", r.Elapsed),
			fmt.Sprintf("%d", r.Latency),
			fmt.Sprintf("%d", r.ConnectTime),
			fmt.Sprintf("%d", r.SentBytes),
			fmt.Sprintf("%d", r.Bytes),
		})
	}
}

func DownloadErrorReport(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID"))
		return
	}

	execution, err := service.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	analysis, err := service.GetExecutionErrors(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error("生成错误报告失败: "+err.Error()))
		return
	}

	content := service.BuildExecutionErrorReportMarkdown(execution, analysis)
	filename := fmt.Sprintf("execution_%d_error_report.md", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, content)
}

// DownloadAll 导出完整结果 ZIP 包
func DownloadAll(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	// 获取执行记录
	execution, err := service.GetExecution(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("execution_%d_full.zip", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/zip")

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// 1. 添加日志文件
	if execution.LogPath != "" {
		addFileToZip(zipWriter, execution.LogPath, "execution.log")
	}

	// 2. 添加 JTL 结果文件
	if execution.ResultPath != "" {
		addFileToZip(zipWriter, execution.ResultPath, "result.jtl")
	}

	// 3. 添加报告目录
	if execution.ReportPath != "" {
		addDirToZip(zipWriter, execution.ReportPath, "report")
	}

	// 4. 生成并添加 summary.json
	if execution.SummaryData != "" {
		summaryData := execution.SummaryData
		if !json.Valid([]byte(summaryData)) {
			// 如果 SummaryData 不是有效的 JSON，包装一下
			summaryData = fmt.Sprintf(`{"execution_id":%d,"script_name":"%s","status":"%s","data":%s}`,
				id, execution.ScriptName, execution.Status, summaryData)
		}
		// 美化 JSON
		var summaryObj map[string]interface{}
		if err := json.Unmarshal([]byte(summaryData), &summaryObj); err == nil {
			prettyJSON, _ := json.MarshalIndent(summaryObj, "", "  ")
			writer, _ := zipWriter.Create("summary.json")
			writer.Write(prettyJSON)
		} else {
			// 直接写入原始数据
			writer, _ := zipWriter.Create("summary.json")
			writer.Write([]byte(summaryData))
		}
	}
}

// addFileToZip 将单个文件添加到 ZIP
func addFileToZip(zipWriter *zip.Writer, filePath, zipPath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// addDirToZip 将目录添加到 ZIP
func addDirToZip(zipWriter *zip.Writer, dirPath, zipPrefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		zipPath := filepath.Join(zipPrefix, relPath)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(zipPath)

		if info.IsDir() {
			header.Name += "/"
			_, err = zipWriter.CreateHeader(header)
			return err
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// GetExecutionLog SSE 流式返回日志
func GetExecutionLog(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	// 获取日志路径
	logPath, err := service.GetExecutionLogPath(id)
	if err != nil {
		c.SSEvent("error", err.Error())
		return
	}

	if c.Query("snapshot") == "1" {
		tail := 300
		if tailStr := c.Query("tail"); tailStr != "" {
			if parsed, parseErr := strconv.Atoi(tailStr); parseErr == nil && parsed > 0 && parsed <= 5000 {
				tail = parsed
			}
		}
		content, readErr := readExecutionLogTail(logPath, tail)
		if readErr != nil {
			c.String(http.StatusOK, "")
			return
		}
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, content)
		return
	}

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	// 先检查执行状态，如果已完成则直接读取并发送 complete
	execution, _ := service.GetExecution(id)
	if execution != nil && execution.Status != "running" {
		// 执行已完成，直接读取日志并发送
		if logPath != "" {
			if content, err := os.ReadFile(logPath); err == nil {
				scanner := bufio.NewScanner(strings.NewReader(string(content)))
				for scanner.Scan() {
					c.SSEvent("message", scanner.Text())
				}
			}
		}
		c.SSEvent("complete", "")
		return
	}

	// 等待日志文件创建，同时监听客户端断开
	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断开连接
			return
		default:
		}

		_, err := os.Stat(logPath)
		if err == nil {
			break
		}
		// 检查执行是否已经结束（可能没有创建日志文件）
		execution, err := service.GetExecution(id)
		if err != nil {
			c.SSEvent("error", "执行记录不存在")
			return
		}
		if execution.Status != "running" {
			c.SSEvent("message", "执行已结束，无日志内容")
			c.SSEvent("complete", "")
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 打开日志文件
	file, err := os.Open(logPath)
	if err != nil {
		c.SSEvent("error", "无法打开日志文件")
		return
	}
	defer file.Close()

	// 创建 flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.SSEvent("error", "不支持流式输出")
		return
	}

	// 使用 bufio.Scanner 逐行读取
	scanner := bufio.NewScanner(file)
	lastPos := int64(0)
	noNewContentCount := 0
	const maxNoContentWait = 300 // 30秒超时（300 * 100ms）

	for {
		select {
		case <-c.Request.Context().Done():
			// 客户端断开连接，立即返回
			return
		default:
		}

		if scanner.Scan() {
			line := scanner.Text()
			c.SSEvent("message", line)
			flusher.Flush()
			lastPos += int64(len(line)) + 1 // +1 for newline
			noNewContentCount = 0
		} else {
			// 检查执行是否完成
			execution, _ := service.GetExecution(id)
			if execution != nil && execution.Status != "running" {
				// 执行完成，读取剩余内容并立即退出
				file.Seek(lastPos, 0)
				scanner = bufio.NewScanner(file)
				for scanner.Scan() {
					line := scanner.Text()
					c.SSEvent("message", line)
					flusher.Flush()
				}
				c.SSEvent("complete", "")
				flusher.Flush()
				return
			}

			// 没有新内容，等待
			noNewContentCount++
			if noNewContentCount > maxNoContentWait {
				c.SSEvent("complete", "timeout")
				flusher.Flush()
				return
			}

			// 使用 select 实现可中断的等待
			select {
			case <-c.Request.Context().Done():
				return
			case <-time.After(100 * time.Millisecond):
				// 重新定位文件，继续读取
				file.Seek(lastPos, 0)
				scanner = bufio.NewScanner(file)
			}
		}
	}
}

func readExecutionLogTail(logPath string, tail int) (string, error) {
	content, err := os.ReadFile(logPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}
	return strings.Join(filtered, "\n"), nil
}

// SetBaselineRequest 设置/取消基准线请求
type SetBaselineRequest struct {
	Action string `json:"action"` // "set" or "unset"
}

// SetBaseline 设置或取消基准线
func SetBaseline(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的ID"))
		return
	}

	var req SetBaselineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的请求参数"))
		return
	}

	var opErr error
	if req.Action == "unset" {
		opErr = service.UnsetBaseline(id)
	} else {
		opErr = service.SetBaseline(id)
	}

	if opErr != nil {
		c.JSON(http.StatusInternalServerError, model.Error(opErr.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{"success": true}))
}

// CompareExecutions 对比两次执行
func CompareExecutions(c *gin.Context) {
	id1, err := strconv.ParseInt(c.Query("id1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID1"))
		return
	}

	id2, err := strconv.ParseInt(c.Query("id2"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID2"))
		return
	}

	result, err := service.CompareExecutions(id1, id2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(result))
}

// collectMasterSystemStats 采集 Master 本地系统指标
func collectMasterSystemStats() string {
	stats := map[string]interface{}{}

	// CPU
	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err == nil && len(cpuPercent) > 0 {
		cpuCount, _ := cpu.Counts(true)
		stats["cpu"] = map[string]interface{}{
			"percent": math.Round(cpuPercent[0]*100) / 100,
			"count":   cpuCount,
		}
	}

	// Memory
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		stats["memory"] = map[string]interface{}{
			"total":   memInfo.Total / 1024 / 1024, // MB
			"used":    memInfo.Used / 1024 / 1024,
			"percent": math.Round(memInfo.UsedPercent*100) / 100,
		}
	}

	// Disk
	diskInfo, err := disk.Usage("/")
	if err == nil {
		stats["disk"] = map[string]interface{}{
			"total":   diskInfo.Total / 1024 / 1024,
			"used":    diskInfo.Used / 1024 / 1024,
			"percent": math.Round(diskInfo.UsedPercent*100) / 100,
		}
	}

	// Network connections
	conns, err := gopsnet.Connections("tcp")
	if err == nil {
		stats["network"] = map[string]interface{}{
			"connections": len(conns),
		}
	}

	jsonBytes, _ := json.Marshal(stats)
	return string(jsonBytes)
}

func collectExecutionNodeMetricsData(execution *model.Execution) ([]gin.H, error) {
	if execution == nil {
		return nil, fmt.Errorf("执行记录不存在")
	}
	if execution.Status != "running" {
		return []gin.H{}, nil
	}

	var slaveIDs []int64
	if strings.TrimSpace(execution.SlaveIDs) != "" {
		if err := json.Unmarshal([]byte(execution.SlaveIDs), &slaveIDs); err != nil {
			slaveIDs = []int64{}
		}
	}

	nodes := []gin.H{}
	masterStats := collectMasterSystemStats()
	masterHost := config.GlobalConfig.JMeter.MasterHostname
	if masterHost == "" {
		masterHost = "localhost"
	}
	masterParticipating := true
	if execution.Diagnostics != nil {
		mode := execution.Diagnostics.Mode
		masterParticipating = mode == "local" || mode == "distributed_with_master" || execution.Diagnostics.IncludeMaster
	}
	nodes = append(nodes, gin.H{
		"id":           0,
		"name":         "Master",
		"host":         masterHost,
		"role":         "master",
		"online":       true,
		"status":       "online",
		"agent_status": "online",
		"participating": masterParticipating,
		"system_stats": masterStats,
	})

	for _, sid := range slaveIDs {
		var slave model.Slave
		var lastCheckTime sql.NullString
		var agentCheckTime sql.NullString
		var systemStats sql.NullString
		var agentStatus sql.NullString
		err := database.DB.QueryRow(
			"SELECT id, name, host, port, status, agent_status, agent_port, agent_token, last_check_time, agent_check_time, system_stats, agent_uptime, created_at FROM slaves WHERE id = ?",
			sid,
		).Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &agentStatus, &slave.AgentPort, &slave.AgentToken, &lastCheckTime, &agentCheckTime, &systemStats, &slave.AgentUptime, &slave.CreatedAt)

		if err != nil {
			// Slave 不存在，跳过
			continue
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

		result, _ := service.CheckSlaveAgent(&slave)
		nodes = append(nodes, gin.H{
			"id":            slave.ID,
			"name":          slave.Name,
			"host":          slave.Host,
			"port":          slave.Port,
			"role":          "slave",
			"online":        result.Online,
			"status":        slave.Status,
			"agent_status":  slave.AgentStatus,
			"agent_uptime":  result.AgentUptime,
			"participating": true,
			"system_stats":  result.SystemStats,
		})
	}

	return nodes, nil
}

// GetExecutionNodeMetrics 获取执行节点的实时系统指标
func GetExecutionNodeMetrics(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的执行ID"))
		return
	}

	execution, err := service.GetExecution(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, model.Error("执行记录不存在"))
		} else {
			c.JSON(http.StatusInternalServerError, model.Error("查询执行记录失败: "+err.Error()))
		}
		return
	}

	nodes, err := collectExecutionNodeMetricsData(execution)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error("获取节点指标失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{"nodes": nodes}))
}
