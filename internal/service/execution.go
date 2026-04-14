package service

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
)

func isTransactionSample(label, url, responseMessage string) bool {
	normalizedURL := strings.TrimSpace(strings.ToLower(url))
	if normalizedURL == "" || normalizedURL == "null" {
		return true
	}
	normalizedLabel := strings.TrimSpace(strings.ToLower(label))
	if strings.Contains(normalizedLabel, "事务控制器") || strings.Contains(normalizedLabel, "transaction") {
		return true
	}
	normalizedMessage := strings.TrimSpace(strings.ToLower(responseMessage))
	return strings.Contains(normalizedMessage, "number of samples in transaction")
}

func isRequestSample(url string) bool {
	normalizedURL := strings.TrimSpace(strings.ToLower(url))
	return normalizedURL != "" && normalizedURL != "null"
}

// 全局进程管理器，用于存储正在执行的命令
type executionProcessGroup struct {
	Commands []*exec.Cmd
	Cancel   func() // 取消超时 timer 的函数
}

var executionProcesses sync.Map

type errorAnalysisCacheEntry struct {
	Signature string
	ExpiresAt time.Time
	Analysis  *ErrorAnalysis
}

type liveMetricsCacheEntry struct {
	Signature string
	ExpiresAt time.Time
	Metrics   *LiveExecutionMetrics
}

var (
	errorAnalysisCache   = make(map[int64]errorAnalysisCacheEntry)
	errorAnalysisCacheMu sync.RWMutex
	liveMetricsCache     = make(map[int64]liveMetricsCacheEntry)
	liveMetricsCacheMu   sync.RWMutex
)

const errorAnalysisCacheTTL = 2 * time.Second
const liveMetricsCacheTTL = 1500 * time.Millisecond

// setProcessGroup 设置进程组属性（仅在 Unix 系统上）
func setProcessGroup(cmd *exec.Cmd) {
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
}

// killProcessGroup 杀死进程组（仅在 Unix 系统上），返回是否成功
func killProcessGroup(cmd *exec.Cmd) bool {
	if runtime.GOOS == "windows" || cmd.Process == nil {
		return false
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return false
	}
	// 负号表示杀进程组
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		return false
	}
	return true
}

// calcJVMArgs 动态计算 JVM 内存参数，取系统可用内存的 80%
func calcJVMArgs() string {
	var totalMB uint64

	// 尝试从 /proc/meminfo 读取可用内存（Linux）
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/proc/meminfo"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "MemAvailable:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
							totalMB = kb / 1024 // KB 转 MB
						}
					}
					break
				}
			}
		}
	}

	// 回退：使用 Go runtime 获取系统总内存
	if totalMB == 0 {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		totalMB = m.Sys / 1024 / 1024
		if totalMB < 1024 {
			totalMB = 4096 // 默认假定 4GB
		}
	}

	// 取 80%，至少 512MB，最多不超过 32GB
	maxHeap := totalMB * 80 / 100
	if maxHeap < 512 {
		maxHeap = 512
	}
	if maxHeap > 32768 {
		maxHeap = 32768
	}

	initHeap := maxHeap / 4
	if initHeap < 256 {
		initHeap = 256
	}

	fmt.Printf("[内存检测] 可用: %dMB, JVM分配: -Xms%dm -Xmx%dm\n", totalMB, initHeap, maxHeap)
	return fmt.Sprintf("-Xms%dm -Xmx%dm", initHeap, maxHeap)
}

// CreateExecution 创建并启动执行
type ExecutionEnvironmentValidationError struct {
	Snapshot *executionEnvironmentSnapshot
}

func (e *ExecutionEnvironmentValidationError) Error() string {
	if e == nil || e.Snapshot == nil || len(e.Snapshot.Warnings) == 0 {
		return "环境一致性存在差异"
	}
	return "环境一致性存在差异：" + strings.Join(e.Snapshot.Warnings, "; ")
}

func CreateExecution(scriptID int64, slaveIDs []int64, remarks string, saveHTTPDetails bool, includeMaster bool, splitCSV bool, ignoreEnvironmentWarnings bool) (*model.Execution, error) {
	// 1. 查询脚本信息
	var script model.Script
	var scriptFilePath string
	err := database.DB.QueryRow(
		"SELECT id, name, file_path FROM scripts WHERE id = ?",
		scriptID,
	).Scan(&script.ID, &script.Name, &scriptFilePath)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("脚本不存在")
		}
		return nil, fmt.Errorf("查询脚本失败: %w", err)
	}

	// 2. 查询选中的 slave 信息
	slaveHosts := []string{}
	var offlineSlaveIDs []int64 // 离线的 slave ID 列表
	slaves := []model.Slave{}   // 存储完整的 slave 信息
	var environmentSnapshot *executionEnvironmentSnapshot
	if len(slaveIDs) > 0 {
		placeholders := make([]string, len(slaveIDs))
		queryArgs := make([]interface{}, len(slaveIDs))
		for i, id := range slaveIDs {
			placeholders[i] = "?"
			queryArgs[i] = id
		}
		query := fmt.Sprintf(
			"SELECT id, name, host, port, agent_port, agent_token FROM slaves WHERE id IN (%s) AND status = 'online'",
			strings.Join(placeholders, ","),
		)
		rows, err := database.DB.Query(query, queryArgs...)
		if err != nil {
			return nil, fmt.Errorf("查询 slave 失败: %w", err)
		}
		defer rows.Close()

		onlineIDs := make(map[int64]bool)
		for rows.Next() {
			var slave model.Slave
			if err := rows.Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.AgentPort, &slave.AgentToken); err != nil {
				fmt.Printf("[警告] 扫描 Slave 数据失败: %v\n", err)
				continue
			}
			onlineIDs[slave.ID] = true
			slaveHosts = append(slaveHosts, fmt.Sprintf("%s:%d", slave.Host, slave.Port))
			slaves = append(slaves, slave)
		}

		// 检查哪些 slave 离线
		for _, id := range slaveIDs {
			if !onlineIDs[id] {
				offlineSlaveIDs = append(offlineSlaveIDs, id)
			}
		}

		if len(slaveHosts) == 0 {
			return nil, fmt.Errorf("选中的 Slave 节点均不在线，请检查节点状态")
		}
	}

	if len(slaveHosts) > 0 {
		masterHost := strings.TrimSpace(config.GlobalConfig.JMeter.MasterHostname)
		if masterHost == "" {
			return nil, fmt.Errorf("分布式执行前请先配置 Master 回调地址")
		}
		callbackProbeURL := fmt.Sprintf("http://%s:%d/api/executions/callback-probe", masterHost, config.GlobalConfig.Server.Port)
		var callbackFailures []string
		for _, slave := range slaves {
			result, err := CheckSlaveCallbackReachability(slave, callbackProbeURL)
			if err != nil {
				callbackFailures = append(callbackFailures, fmt.Sprintf("%s(%s): %v", slave.Name, slave.Host, err))
				continue
			}
			if !result.Reachable {
				reason := result.Error
				if reason == "" {
					reason = fmt.Sprintf("HTTP %d", result.StatusCode)
				}
				callbackFailures = append(callbackFailures, fmt.Sprintf("%s(%s): %s", slave.Name, slave.Host, reason))
			}
		}
		if len(callbackFailures) > 0 {
			return nil, fmt.Errorf("分布式回调可达性预检失败，请确认 Slave 可回调 Master：%s", strings.Join(callbackFailures, "; "))
		}
		fmt.Printf("[分布式预检] 回调可达性通过，回调地址: %s\n", callbackProbeURL)

		environmentSnapshot, err = validateExecutionEnvironments(slaves, includeMaster)
		if err != nil {
			return nil, err
		}
		if environmentSnapshot != nil {
			if len(environmentSnapshot.Warnings) > 0 {
				fmt.Printf("[分布式预检] 环境存在差异，基线节点: %s\n", environmentSnapshot.Baseline.Node)
				if !ignoreEnvironmentWarnings {
					return nil, &ExecutionEnvironmentValidationError{Snapshot: environmentSnapshot}
				}
				fmt.Printf("[分布式预检] 已忽略环境差异并继续执行: %s\n", strings.Join(environmentSnapshot.Warnings, "; "))
			} else {
				fmt.Printf("[分布式预检] 环境一致性通过，基线节点: %s\n", environmentSnapshot.Baseline.Node)
			}
		}
	}

	// 3. 创建执行记录
	startTime := time.Now().Format("2006-01-02 15:04:05")
	slaveIDsJSON, _ := json.Marshal(slaveIDs)

	result, err := database.DB.Exec(
		"INSERT INTO executions (script_id, script_name, slave_ids, status, start_time, remarks, save_http_details, include_master, split_csv, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		scriptID, script.Name, string(slaveIDsJSON), "running", startTime, remarks, boolToInt(saveHTTPDetails), boolToInt(includeMaster), boolToInt(splitCSV), startTime,
	)
	if err != nil {
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	execID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取执行ID失败: %w", err)
	}

	// 4. 创建结果目录
	resultDir := filepath.Join(config.GlobalConfig.Dirs.Results, fmt.Sprintf("%d", execID))
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return nil, fmt.Errorf("创建结果目录失败: %w", err)
	}
	if err := saveExecutionEnvironmentSnapshot(resultDir, environmentSnapshot); err != nil {
		fmt.Printf("[警告] 保存环境快照失败: %v\n", err)
	}

	resultPath := filepath.Join(resultDir, "result.jtl")
	localResultPath := filepath.Join(resultDir, "result-local.jtl")
	remoteResultPath := filepath.Join(resultDir, "result-remote.jtl")
	reportPath := filepath.Join(resultDir, "report")
	logPath := filepath.Join(resultDir, "execution.log")
	errorDetailPath := filepath.Join(resultDir, "error-details.ndjson")
	errorDetailUploadEnabled := false
	errorDetailUploadURL := ""
	errorDetailUploadToken := ""

	// 更新记录的路径信息
	_, err = database.DB.Exec(
		"UPDATE executions SET result_path = ?, report_path = ?, log_path = ? WHERE id = ?",
		resultPath, reportPath, logPath, execID,
	)
	if err != nil {
		return nil, fmt.Errorf("更新执行记录路径失败: %w", err)
	}

	localScriptPath := scriptFilePath  // Master 本地执行用的脚本
	remoteScriptPath := scriptFilePath // Slave 远程执行用的脚本
	fmt.Printf("[执行 #%d] save_http_details=%t\n", execID, saveHTTPDetails)
	runDistributedWithLocal := len(slaveHosts) > 0 && includeMaster
	fmt.Printf("[执行 #%d] include_master=%t\n", execID, includeMaster)
	scriptDir := filepath.Dir(scriptFilePath)
	jmxContent, err := os.ReadFile(scriptFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取JMX文件失败: %w", err)
	}

	attachedFileLookup := make(map[string]string)
	if scriptFiles, fileErr := GetScriptFiles(scriptID); fileErr == nil {
		for _, file := range scriptFiles {
			base := filepath.Base(file.FileName)
			if base == "" {
				base = filepath.Base(file.FilePath)
			}
			if base != "" {
				attachedFileLookup[base] = file.FilePath
			}
		}
	}

	// === CSV 拆分分发（仅分布式 + 开启 split_csv 时） ===
	successfullySplitCSVRefs := make(map[string]string)
	if splitCSV && len(slaves) > 0 {
		csvRefs := extractCSVDataSetReferences(string(jmxContent))

		if len(csvRefs) > 0 {
			csvRefGroups := make(map[string][]csvDataSetReference)
			csvFiles := make([]string, 0)
			for _, ref := range csvRefs {
				if _, exists := csvRefGroups[ref.Filename]; !exists {
					csvFiles = append(csvFiles, ref.Filename)
				}
				csvRefGroups[ref.Filename] = append(csvRefGroups[ref.Filename], ref)
			}

			fmt.Printf("[执行 #%d] 发现CSV文件: %v\n", execID, csvFiles)

			partCount := len(slaves)
			if includeMaster {
				partCount++
			}

			csvDataDir := config.GlobalConfig.JMeter.AgentCSVDataDir
			localCSVDir := filepath.Join(resultDir, "csv-data")
			os.MkdirAll(localCSVDir, 0755)

			var allPartFiles []string
			var cleanupCSVTargetNames []string

			for index, csvFile := range csvFiles {
				csvFileName := filepath.Base(csvFile)
				targetName := buildRuntimeTargetName(execID, "csv", csvFile, index+1)

				originalPath, resolveErr := resolveRuntimeDependencySourcePath(csvFile, scriptDir, attachedFileLookup)
				if resolveErr != nil {
					fmt.Printf("[警告] 解析CSV文件失败 %s: %v\n", csvFileName, resolveErr)
					continue
				}

				hasHeader, headerConsistent := hasConsistentCSVHeaderConfig(csvRefGroups[csvFile])
				if !headerConsistent {
					fmt.Printf("[警告] CSV %s 在多个 CSVDataSet 中 ignoreFirstLine 配置不一致，将回退为常规依赖文件分发逻辑\n", csvFileName)
					continue
				}

				parts, err := SplitCSV(originalPath, partCount, hasHeader, resultDir, targetName)
				if err != nil {
					fmt.Printf("[警告] 拆分CSV文件失败 %s: %v\n", csvFileName, err)
					continue
				}
				allPartFiles = append(allPartFiles, parts...)
				fmt.Printf("[执行 #%d] CSV %s 已拆分为 %d 份\n", execID, csvFileName, len(parts))

				distributedSuccessfully := true
				for i, slave := range slaves {
					if i >= len(parts) {
						break
					}
					client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
					if err := client.UploadFile(parts[i], targetName); err != nil {
						fmt.Printf("[警告] 上传CSV到Slave %s失败: %v\n", slave.Host, err)
						distributedSuccessfully = false
					} else {
						fmt.Printf("[执行 #%d] 已上传 %s 到 Slave %s\n", execID, targetName, slave.Host)
					}
				}

				if includeMaster && len(parts) > len(slaves) {
					masterPart := parts[len(slaves)]
					localPath := filepath.Join(localCSVDir, targetName)
					if err := copyFile(masterPart, localPath); err != nil {
						fmt.Printf("[警告] 复制CSV到本地失败 %s: %v\n", csvFileName, err)
						distributedSuccessfully = false
					} else {
						fmt.Printf("[执行 #%d] 已复制 %s 到本地\n", execID, targetName)
					}
					allPartFiles = append(allPartFiles, localPath)
				}

				if distributedSuccessfully {
					successfullySplitCSVRefs[csvFile] = targetName
					cleanupCSVTargetNames = append(cleanupCSVTargetNames, targetName)
				} else {
					fmt.Printf("[警告] CSV %s 未完整分发成功，将回退为常规依赖文件分发逻辑\n", csvFileName)
				}
			}

			if len(successfullySplitCSVRefs) > 0 {
				runtimeRemoteJMXPath := filepath.Join(resultDir, "runtime-remote.jmx")
				remoteContent := replaceCSVDataSetPathsWithMap(string(jmxContent), csvDataDir, successfullySplitCSVRefs)
				if err := os.WriteFile(runtimeRemoteJMXPath, []byte(remoteContent), 0644); err != nil {
					fmt.Printf("[警告] 写入 Slave 运行时JMX失败: %v\n", err)
				} else {
					remoteScriptPath = runtimeRemoteJMXPath
					fmt.Printf("[执行 #%d] 已生成 Slave 运行时JMX: %s\n", execID, runtimeRemoteJMXPath)
				}

				if includeMaster {
					localCSVAbsPath, err := filepath.Abs(localCSVDir)
					if err != nil {
						fmt.Printf("[警告] 获取本地CSV绝对路径失败: %v\n", err)
						localCSVAbsPath = localCSVDir
					}
					runtimeLocalJMXPath := filepath.Join(resultDir, "runtime-local.jmx")
					localContent := replaceCSVDataSetPathsWithMap(string(jmxContent), localCSVAbsPath, successfullySplitCSVRefs)
					if err := os.WriteFile(runtimeLocalJMXPath, []byte(localContent), 0644); err != nil {
						fmt.Printf("[警告] 写入 Master 运行时JMX失败: %v\n", err)
					} else {
						localScriptPath = runtimeLocalJMXPath
						fmt.Printf("[执行 #%d] 已生成 Master 运行时JMX: %s (CSV路径: %s)\n", execID, runtimeLocalJMXPath, localCSVAbsPath)
					}
				}
			}

			go func(execID int64, slaves []model.Slave, csvFiles []string) {
				for {
					var status string
					if err := database.DB.QueryRow(
						"SELECT status FROM executions WHERE id = ?", execID,
					).Scan(&status); err != nil {
						fmt.Printf("[警告] 查询执行 #%d 状态失败: %v\n", execID, err)
						time.Sleep(2 * time.Second)
						continue
					}
					if status != "running" {
						break
					}
					time.Sleep(2 * time.Second)
				}

				for _, slave := range slaves {
					client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
					for _, csvFile := range csvFiles {
						if err := client.DeleteFile(csvFile); err != nil {
							fmt.Printf("[警告] 清理Slave %s上的CSV失败 %s: %v\n", slave.Host, csvFile, err)
						}
					}
				}

				for _, partFile := range allPartFiles {
					os.Remove(partFile)
				}
				os.RemoveAll(localCSVDir)

				fmt.Printf("[执行 #%d] CSV分发清理完成\n", execID)
			}(execID, slaves, cleanupCSVTargetNames)
		}
	}

	if len(slaveHosts) > 0 {
		attachedNames := getAttachedScriptFileNames(scriptID)
		dependencyScan := inspectJMXDependencies(scriptFilePath, attachedNames, true, splitCSV)
		splitCSVSet := make(map[string]bool)
		if splitCSV {
			for dep := range successfullySplitCSVRefs {
				splitCSVSet[dep] = true
			}
		}

		remoteDependencyMap := make(map[string]string)
		cleanupDependencyTargetNames := make([]string, 0)
		for depIndex, dep := range append(append([]string{}, dependencyScan.CSVFiles...), dependencyScan.FileDependencies...) {
			baseName := filepath.Base(dep)
			if baseName == "" || splitCSVSet[dep] {
				continue
			}

			sourcePath, resolveErr := resolveRuntimeDependencySourcePath(dep, scriptDir, attachedFileLookup)
			if resolveErr != nil {
				fmt.Printf("[警告] 解析依赖文件失败 %s: %v\n", dep, resolveErr)
				continue
			}

			targetName := buildRuntimeTargetName(execID, "dep", dep, depIndex+1)
			uploaded := false
			for _, slave := range slaves {
				client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
				if err := client.UploadFile(sourcePath, targetName); err != nil {
					fmt.Printf("[警告] 上传依赖到Slave %s失败 %s: %v\n", slave.Host, targetName, err)
				} else {
					uploaded = true
					fmt.Printf("[执行 #%d] 已上传依赖 %s 到 Slave %s\n", execID, targetName, slave.Host)
				}
			}

			if uploaded {
				remoteDependencyMap[dep] = targetName
				cleanupDependencyTargetNames = append(cleanupDependencyTargetNames, targetName)
			}
		}

		if len(remoteDependencyMap) > 0 {
			baseContentPath := remoteScriptPath
			baseContentBytes, readErr := os.ReadFile(baseContentPath)
			if readErr != nil {
				fmt.Printf("[警告] 读取远端运行脚本失败 %s: %v\n", baseContentPath, readErr)
			} else {
				rewrittenContent := replaceFileDependencyPathsWithMap(string(baseContentBytes), config.GlobalConfig.JMeter.AgentCSVDataDir, remoteDependencyMap)
				runtimeRemoteJMXPath := filepath.Join(resultDir, "runtime-remote.jmx")
				if err := os.WriteFile(runtimeRemoteJMXPath, []byte(rewrittenContent), 0644); err != nil {
					fmt.Printf("[警告] 写入带依赖路径的Slave运行时JMX失败: %v\n", err)
				} else {
					remoteScriptPath = runtimeRemoteJMXPath
					fmt.Printf("[执行 #%d] 已生成带依赖路径的 Slave 运行时JMX: %s\n", execID, runtimeRemoteJMXPath)
				}
			}

			go func(execID int64, slaves []model.Slave, fileNames []string) {
				for {
					var status string
					if err := database.DB.QueryRow("SELECT status FROM executions WHERE id = ?", execID).Scan(&status); err != nil {
						time.Sleep(2 * time.Second)
						continue
					}
					if status != "running" {
						break
					}
					time.Sleep(2 * time.Second)
				}

				for _, slave := range slaves {
					client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
					for _, fileName := range fileNames {
						if err := client.DeleteFile(fileName); err != nil {
							fmt.Printf("[警告] 清理Slave %s上的依赖失败 %s: %v\n", slave.Host, fileName, err)
						}
					}
				}
			}(execID, slaves, dedupeStrings(cleanupDependencyTargetNames))
		}
	}

	if saveHTTPDetails {
		if len(slaveHosts) > 0 {
			runtimeRemoteScriptPath := filepath.Join(resultDir, "runtime-remote-with-details.jmx")
			if err := createRuntimeJMXWithErrorDetailListener(remoteScriptPath, runtimeRemoteScriptPath); err != nil {
				return nil, fmt.Errorf("生成分布式错误明细运行脚本失败: %w", err)
			}
			remoteScriptPath = runtimeRemoteScriptPath
			fmt.Printf("[执行 #%d] 已生成分布式错误明细运行脚本: %s\n", execID, runtimeRemoteScriptPath)

			if includeMaster {
				runtimeLocalScriptPath := filepath.Join(resultDir, "runtime-local-with-details.jmx")
				if err := createRuntimeJMXWithErrorDetailListener(localScriptPath, runtimeLocalScriptPath); err != nil {
					return nil, fmt.Errorf("生成本地错误明细运行脚本失败: %w", err)
				}
				localScriptPath = runtimeLocalScriptPath
				fmt.Printf("[执行 #%d] 已生成本地错误明细运行脚本: %s\n", execID, runtimeLocalScriptPath)
			}
		} else {
			runtimeLocalScriptPath := filepath.Join(resultDir, "runtime-with-details.jmx")
			if err := createRuntimeJMXWithErrorDetailListener(localScriptPath, runtimeLocalScriptPath); err != nil {
				return nil, fmt.Errorf("生成带错误明细监听器的运行脚本失败: %w", err)
			}
			localScriptPath = runtimeLocalScriptPath
			fmt.Printf("[执行 #%d] 已生成错误明细运行脚本: %s\n", execID, runtimeLocalScriptPath)
		}
		fmt.Printf("[执行 #%d] 本地错误明细文件: %s\n", execID, errorDetailPath)
		if len(slaveHosts) > 0 {
			masterHost := strings.TrimSpace(config.GlobalConfig.JMeter.MasterHostname)
			if masterHost == "" {
				return nil, fmt.Errorf("分布式保存 HTTP 明细需要先配置 Master 回调地址")
			}
			token, err := generateExecutionUploadToken()
			if err != nil {
				return nil, fmt.Errorf("生成错误明细上传令牌失败: %w", err)
			}
			if err := saveExecutionUploadToken(resultDir, token); err != nil {
				return nil, fmt.Errorf("保存错误明细上传令牌失败: %w", err)
			}
			errorDetailUploadEnabled = true
			errorDetailUploadToken = token
			errorDetailUploadURL = fmt.Sprintf("http://%s:%d/api/executions/%d/error-details/upload", masterHost, config.GlobalConfig.Server.Port, execID)
			fmt.Printf("[执行 #%d] 分布式错误明细回传: %s\n", execID, errorDetailUploadURL)
		}
	}

	// 5. 构建 JMeter 命令
	jmeterPath := config.GlobalConfig.JMeter.Path
	if jmeterPath == "" {
		jmeterPath = "jmeter"
	}

	// 使用属性文件 + CLI 属性双保险，尽量确保本地和分布式节点都写出请求/响应详情字段。
	saveServiceProps := map[string]string{
		"jmeter.save.saveservice.output_format":                     "csv",
		"jmeter.save.saveservice.print_field_names":                 "true",
		"jmeter.save.saveservice.request_headers":                   "true",
		"jmeter.save.saveservice.response_data":                     "true",
		"jmeter.save.saveservice.response_data.on_error":            "true",
		"jmeter.save.saveservice.response_headers":                  "true",
		"jmeter.save.saveservice.samplerData":                       "true",
		"jmeter.save.saveservice.url":                               "true",
		"jmeter.save.saveservice.encoding":                          "true",
		"jmeter.save.saveservice.sent_bytes":                        "true",
		"jmeter.save.saveservice.latency":                           "true",
		"jmeter.save.saveservice.connect_time":                      "true",
		"jmeter.save.saveservice.assertion_results_failure_message": "true",
	}
	propKeys := make([]string, 0, len(saveServiceProps))
	for key := range saveServiceProps {
		propKeys = append(propKeys, key)
	}
	sort.Strings(propKeys)
	propLines := []string{"# JMeter 结果保存配置 - 由 jmeter-admin 自动生成"}
	for _, key := range propKeys {
		propLines = append(propLines, fmt.Sprintf("%s=%s", key, saveServiceProps[key]))
	}
	propsContent := strings.Join(propLines, "\n") + "\n"
	propsFile := filepath.Join(resultDir, "jmeter.properties")
	if err := os.WriteFile(propsFile, []byte(propsContent), 0644); err != nil {
		fmt.Printf("[警告] 创建临时属性文件失败: %v\n", err)
	}

	buildBaseArgs := func(targetResultPath string, generateReport bool, scriptPath string) []string {
		args := []string{
			"-n", "-t", scriptPath,
			"-l", targetResultPath,
			"-q", propsFile,
		}
		if generateReport {
			args = append(args, "-e", "-o", reportPath)
		}
		for _, key := range propKeys {
			args = append(args, "-J"+key+"="+saveServiceProps[key])
		}
		return args
	}

	buildLocalArgs := func(targetResultPath string, generateReport bool, scriptPath string) []string {
		args := buildBaseArgs(targetResultPath, generateReport, scriptPath)
		if saveHTTPDetails {
			args = append(args, "-JjmeterAdmin.errorDetailFile="+errorDetailPath)
			args = append(args, "-JjmeterAdmin.captureHttpDetails=true")
			args = append(args, "-JjmeterAdmin.errorDetailUploadEnabled=false")
			args = append(args, "-JjmeterAdmin.detailSource=master-local")
		}
		args = append(args, "-Djmeter.reportgenerator.overall_granularity=1000")
		return args
	}

	buildRemoteArgs := func(targetResultPath string, generateReport bool, scriptPath string) []string {
		args := buildBaseArgs(targetResultPath, generateReport, scriptPath)
		args = append(args, "-R", strings.Join(slaveHosts, ","))
		for _, key := range propKeys {
			args = append(args, "-G"+key+"="+saveServiceProps[key])
		}
		if saveHTTPDetails {
			args = append(args, "-JjmeterAdmin.errorDetailFile="+errorDetailPath)
			args = append(args, "-JjmeterAdmin.captureHttpDetails=true")
			args = append(args, "-JjmeterAdmin.errorDetailUploadEnabled="+strconv.FormatBool(errorDetailUploadEnabled))
			if errorDetailUploadURL != "" {
				args = append(args, "-JjmeterAdmin.errorDetailUploadUrl="+errorDetailUploadURL)
			}
			if errorDetailUploadToken != "" {
				args = append(args, "-JjmeterAdmin.errorDetailUploadToken="+errorDetailUploadToken)
			}
			args = append(args, "-GjmeterAdmin.errorDetailFile="+errorDetailPath)
			args = append(args, "-GjmeterAdmin.captureHttpDetails=true")
			args = append(args, "-GjmeterAdmin.errorDetailUploadEnabled="+strconv.FormatBool(errorDetailUploadEnabled))
			if errorDetailUploadURL != "" {
				args = append(args, "-GjmeterAdmin.errorDetailUploadUrl="+errorDetailUploadURL)
			}
			if errorDetailUploadToken != "" {
				args = append(args, "-GjmeterAdmin.errorDetailUploadToken="+errorDetailUploadToken)
			}
		}
		args = append(args, "-Dserver.rmi.ssl.disable=true")
		args = append(args, "-Djmeter.engine.start.wait=5000")
		args = append(args, "-Djmeter.reportgenerator.overall_granularity=1000")
		if masterHost := config.GlobalConfig.JMeter.MasterHostname; masterHost != "" {
			args = append(args, "-Djava.rmi.server.hostname="+masterHost)
			fmt.Printf("[RMI] Master hostname: %s\n", masterHost)
		}
		return args
	}

	var localArgs []string
	var remoteArgs []string
	runPlan := buildExecutionRunPlan(len(slaveHosts) > 0, includeMaster)
	switch {
	case runPlan.RunLocal && runPlan.RunRemote:
		localArgs = buildLocalArgs(localResultPath, false, localScriptPath)
		remoteArgs = buildRemoteArgs(remoteResultPath, false, remoteScriptPath)
	case runPlan.RunRemote:
		remoteArgs = buildRemoteArgs(resultPath, true, remoteScriptPath)
	default:
		localArgs = buildLocalArgs(resultPath, true, localScriptPath)
	}

	// 输出执行日志
	if len(slaveHosts) > 0 {
		fmt.Printf("[执行 #%d] 分布式模式，Slave: %v\n", execID, slaveHosts)
		if runDistributedWithLocal {
			fmt.Printf("[执行 #%d] 分布式模式包含 Master 本地执行\n", execID)
		}
		if len(offlineSlaveIDs) > 0 {
			fmt.Printf("[警告] 部分 Slave 离线(ID: %v)，将使用在线节点继续执行\n", offlineSlaveIDs)
		}
	} else if len(slaveIDs) == 0 {
		fmt.Printf("[执行 #%d] 本地模式\n", execID)
	}
	// 动态计算 JVM 内存：取系统可用内存的 80%，至少 512MB
	jvmArgs := calcJVMArgs()
	if envJVM := os.Getenv("JVM_ARGS"); envJVM != "" {
		jvmArgs = envJVM
	}
	fmt.Printf("[执行 #%d] JVM_ARGS: %s\n", execID, jvmArgs)

	// 6. 启动 goroutine 异步执行（带 4 小时超时保护）
	go func() {
		// 创建日志文件（必须在 timeoutTimer 之前，因为回调需要引用它）
		logFile, err := os.Create(logPath)
		if err != nil {
			updateExecutionStatus(execID, "failed", "")
			return
		}
		defer logFile.Close()

		// 构建命令列表（必须在 timeoutTimer 之前，因为回调需要引用它）
		commands := make([]*exec.Cmd, 0, 2)
		logPrefix := fmt.Sprintf("[执行 #%d]", execID)
		if len(localArgs) > 0 {
			fmt.Printf("%s 本地命令: %s %s\n", logPrefix, jmeterPath, strings.Join(localArgs, " "))
			cmd := exec.Command(jmeterPath, localArgs...)
			cmd.Env = append(os.Environ(), "JVM_ARGS="+jvmArgs)
			setProcessGroup(cmd)
			commands = append(commands, cmd)
		}
		if len(remoteArgs) > 0 {
			fmt.Printf("%s 分布式命令: %s %s\n", logPrefix, jmeterPath, strings.Join(remoteArgs, " "))
			cmd := exec.Command(jmeterPath, remoteArgs...)
			cmd.Env = append(os.Environ(), "JVM_ARGS="+jvmArgs)
			setProcessGroup(cmd)
			commands = append(commands, cmd)
		}

		// 超时保护：使用 AfterFunc，在超时时杀进程
		var timedOut int32 // atomic 标志位
		timeoutTimer := time.AfterFunc(4*time.Hour, func() {
			if atomic.CompareAndSwapInt32(&timedOut, 0, 1) {
				fmt.Fprintf(logFile, "[执行 #%d] 执行超时（超过4小时），强制终止\n", execID)
				for _, cmd := range commands {
					killProcessGroup(cmd)
				}
				endTime := time.Now().Format("2006-01-02 15:04:05")
				database.DB.Exec(
					"UPDATE executions SET status = ?, end_time = ?, remarks = ? WHERE id = ?",
					"timeout", endTime, "执行超时（超过4小时）", execID,
				)
			}
		})

		processGroup := &executionProcessGroup{
			Commands: commands,
			Cancel: func() {
				timeoutTimer.Stop()
			},
		}
		executionProcesses.Store(execID, processGroup)
		defer executionProcesses.Delete(execID)
		indexerStop := make(chan struct{})
		go startExecutionErrorAnalysisIndexer(execID, indexerStop, logFile)
		defer close(indexerStop)

		var execErr error

		runCommand := func(cmd *exec.Cmd) error {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			return cmd.Run()
		}

		// 直接在当前 goroutine 中执行（不需要内层 goroutine）
		switch len(commands) {
		case 0:
			execErr = fmt.Errorf("未生成可执行的 JMeter 命令")
		case 1:
			execErr = runCommand(commands[0])
		default:
			// 分布式+本地并行执行
			var localErr, remoteErr error
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				localErr = runCommand(commands[0])
			}()
			go func() {
				defer wg.Done()
				remoteErr = runCommand(commands[1])
			}()
			wg.Wait()

			// JTL 合并、报告生成等后续处理
			if mergeErr := mergeJTLFiles([]string{localResultPath, remoteResultPath}, resultPath); mergeErr != nil {
				fmt.Fprintf(logFile, "[执行 #%d] 合并本地与分布式结果失败: %v\n", execID, mergeErr)
				if execErr == nil {
					execErr = mergeErr
				}
			} else {
				reportArgs := []string{"-g", resultPath, "-o", reportPath}
				reportCmd := exec.Command(jmeterPath, reportArgs...)
				reportCmd.Env = append(os.Environ(), "JVM_ARGS="+jvmArgs)
				setProcessGroup(reportCmd)
				reportCmd.Stdout = logFile
				reportCmd.Stderr = logFile
				if reportErr := reportCmd.Run(); reportErr != nil {
					fmt.Fprintf(logFile, "[执行 #%d] 生成合并报告失败: %v\n", execID, reportErr)
				}
			}
			if localErr != nil || remoteErr != nil {
				errMessages := make([]string, 0, 2)
				if localErr != nil {
					errMessages = append(errMessages, "本地执行失败: "+localErr.Error())
				}
				if remoteErr != nil {
					errMessages = append(errMessages, "分布式执行失败: "+remoteErr.Error())
				}
				execErr = fmt.Errorf("%s", strings.Join(errMessages, "; "))
			}
		}

		// 停止超时 timer（如果还没触发）
		timeoutTimer.Stop()

		// 检查是否已被超时标记
		if atomic.LoadInt32(&timedOut) == 1 {
			return // 超时已处理，不再更新状态
		}

		// 7. 命令完成后解析结果
		var summaryData string
		if summary, summaryErr := parseJTLResults(resultPath); summaryErr == nil {
			summaryData = summary
		} else if runDistributedWithLocal {
			summaryData, _ = parseJTLResults(localResultPath, remoteResultPath)
		}

		// 更新执行状态
		status := "success"
		if execErr != nil {
			status = "failed"
		}
		updateExecutionStatus(execID, status, summaryData)
	}()

	// 返回执行记录
	execution := &model.Execution{
		ID:              execID,
		ScriptID:        scriptID,
		ScriptName:      script.Name,
		SlaveIDs:        string(slaveIDsJSON),
		Status:          "running",
		StartTime:       startTime,
		Remarks:         remarks,
		SaveHTTPDetails: saveHTTPDetails,
		IncludeMaster:   includeMaster,
		SplitCSV:        splitCSV,
		ResultPath:      resultPath,
		ReportPath:      reportPath,
		LogPath:         logPath,
		CreatedAt:       startTime,
	}

	return execution, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// updateExecutionStatus 更新执行状态
func updateExecutionStatus(execID int64, status, summaryData string) {
	endTime := time.Now().Format("2006-01-02 15:04:05")

	// 查询开始时间，计算执行时长
	var startTimeStr string
	err := database.DB.QueryRow("SELECT start_time FROM executions WHERE id = ?", execID).Scan(&startTimeStr)
	var duration int64
	if err == nil {
		startTime, parseErr := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if parseErr == nil {
			duration = int64(time.Since(startTime).Seconds())
		}
	}

	_, _ = database.DB.Exec(
		"UPDATE executions SET status = ?, end_time = ?, duration = ?, summary_data = ? WHERE id = ?",
		status, endTime, duration, summaryData, execID,
	)
}

// ListExecutions 分页查询执行记录
func ListExecutions(page, pageSize int) ([]model.Execution, int64, error) {
	return ListExecutionsFiltered(page, pageSize, "", "", "", "", "")
}

// ListExecutionsFiltered 分页查询执行记录（支持筛选）
func ListExecutionsFiltered(page, pageSize int, scriptID, status, keyword, startDate, endDate string) ([]model.Execution, int64, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 构建查询条件
	whereClauses := []string{}
	args := []interface{}{}

	if scriptID != "" {
		whereClauses = append(whereClauses, "script_id = ?")
		args = append(args, scriptID)
	}
	if status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, status)
	}
	if keyword != "" {
		whereClauses = append(whereClauses, "remarks LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	if startDate != "" {
		whereClauses = append(whereClauses, "created_at >= ?")
		args = append(args, startDate)
	}
	if endDate != "" {
		whereClauses = append(whereClauses, "created_at <= ?")
		args = append(args, endDate)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// 查询总数
	var total int64
	countQuery := "SELECT COUNT(*) FROM executions " + whereClause
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %w", err)
	}

	// 查询列表
	query := `SELECT id, script_id, script_name, slave_ids, status, start_time, end_time, duration, remarks, save_http_details, include_master, split_csv, result_path, report_path, summary_data, log_path, is_baseline, created_at FROM executions ` + whereClause + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询执行列表失败: %w", err)
	}
	defer rows.Close()

	var executions []model.Execution
	for rows.Next() {
		var e model.Execution
		var endTime, summaryData, remarks sql.NullString
		var duration sql.NullInt64
		var isBaseline, saveHTTPDetails, includeMaster, splitCSV int
		err := rows.Scan(
			&e.ID, &e.ScriptID, &e.ScriptName, &e.SlaveIDs, &e.Status,
			&e.StartTime, &endTime, &duration, &remarks, &saveHTTPDetails, &includeMaster, &splitCSV, &e.ResultPath, &e.ReportPath,
			&summaryData, &e.LogPath, &isBaseline, &e.CreatedAt,
		)
		if err != nil {
			continue
		}
		if endTime.Valid {
			e.EndTime = endTime.String
		}
		if duration.Valid {
			e.Duration = duration.Int64
		}
		if remarks.Valid {
			e.Remarks = remarks.String
		}
		if summaryData.Valid {
			e.SummaryData = summaryData.String
		}
		e.SaveHTTPDetails = saveHTTPDetails == 1
		e.IncludeMaster = includeMaster == 1
		e.SplitCSV = splitCSV == 1
		e.IsBaseline = isBaseline == 1
		enrichExecutionForDisplay(&e, false)
		executions = append(executions, e)
	}

	return executions, total, nil
}

// ExecutionStats 执行统计数据
type ExecutionStats struct {
	Total     int64 `json:"total"`
	Running   int64 `json:"running"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Stopped   int64 `json:"stopped"`
}

// GetExecutionStats 获取执行统计数据
func GetExecutionStats() (*ExecutionStats, error) {
	stats := &ExecutionStats{}

	err := database.DB.QueryRow("SELECT COUNT(*) FROM executions").Scan(&stats.Total)
	if err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = ?", "running").Scan(&stats.Running)
	if err != nil {
		return nil, fmt.Errorf("查询运行中数量失败: %w", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = ?", "success").Scan(&stats.Completed)
	if err != nil {
		return nil, fmt.Errorf("查询已完成数量失败: %w", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = ?", "failed").Scan(&stats.Failed)
	if err != nil {
		return nil, fmt.Errorf("查询失败数量失败: %w", err)
	}

	err = database.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE status = ?", "stopped").Scan(&stats.Stopped)
	if err != nil {
		return nil, fmt.Errorf("查询已停止数量失败: %w", err)
	}

	return stats, nil
}

// GetExecution 获取执行详情
func GetExecution(id int64) (*model.Execution, error) {
	var e model.Execution
	var endTime, summaryData, remarks sql.NullString
	var duration sql.NullInt64
	var isBaseline, saveHTTPDetails, includeMaster, splitCSV int
	err := database.DB.QueryRow(
		"SELECT id, script_id, script_name, slave_ids, status, start_time, end_time, duration, remarks, save_http_details, include_master, split_csv, result_path, report_path, summary_data, log_path, is_baseline, created_at FROM executions WHERE id = ?",
		id,
	).Scan(
		&e.ID, &e.ScriptID, &e.ScriptName, &e.SlaveIDs, &e.Status,
		&e.StartTime, &endTime, &duration, &remarks, &saveHTTPDetails, &includeMaster, &splitCSV, &e.ResultPath, &e.ReportPath,
		&summaryData, &e.LogPath, &isBaseline, &e.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("执行记录不存在")
		}
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	if endTime.Valid {
		e.EndTime = endTime.String
	}
	if duration.Valid {
		e.Duration = duration.Int64
	}
	if remarks.Valid {
		e.Remarks = remarks.String
	}
	if summaryData.Valid {
		e.SummaryData = summaryData.String
	}
	e.SaveHTTPDetails = saveHTTPDetails == 1
	e.IncludeMaster = includeMaster == 1
	e.SplitCSV = splitCSV == 1
	e.IsBaseline = isBaseline == 1
	enrichExecutionForDisplay(&e, true)

	return &e, nil
}

// GetExecutionLiveMetrics 读取执行中的 JTL，返回实时趋势与摘要
func GetExecutionLiveMetrics(id int64) (*LiveExecutionMetrics, error) {
	execution, err := GetExecution(id)
	if err != nil {
		return nil, err
	}

	resultPath := execution.ResultPath
	if resultPath == "" {
		return &LiveExecutionMetrics{Status: execution.Status, Points: []LiveMetricPoint{}}, nil
	}

	resultPaths := discoverExecutionResultPaths(resultPath)
	if len(resultPaths) == 0 {
		return &LiveExecutionMetrics{Status: execution.Status, Points: []LiveMetricPoint{}}, nil
	}

	cacheSignature := buildLiveMetricsSignature(execution.Status, resultPaths)
	if cached := getCachedLiveMetrics(id, cacheSignature); cached != nil {
		return cached, nil
	}

	colIndex := make(map[string]int)
	headersLoaded := false

	required := []string{"timeStamp", "elapsed", "success"}

	getField := func(record []string, field string) string {
		if idx, ok := colIndex[field]; ok && idx < len(record) {
			return record[idx]
		}
		return ""
	}

	buckets := make(map[int64]*liveBucket)
	var bucketOrder []int64
	totalRequests := 0
	successRequests := 0
	errorRequests := 0
	totalTransactions := 0
	successTransactions := 0
	errorTransactions := 0
	totalRT := int64(0)
	totalTransactionRT := int64(0)
	minTs := int64(0)
	maxEndTs := int64(0)

	processRecord := func(record []string) {
		ts, err := strconv.ParseInt(getField(record, "timeStamp"), 10, 64)
		if err != nil {
			return
		}
		elapsed, err := strconv.ParseInt(getField(record, "elapsed"), 10, 64)
		if err != nil {
			return
		}

		second := ts / 1000
		bucket, exists := buckets[second]
		if !exists {
			bucket = &liveBucket{}
			buckets[second] = bucket
			bucketOrder = append(bucketOrder, second)
		}

		label := getField(record, "label")
		url := getField(record, "URL")
		responseMessage := getField(record, "responseMessage")
		success := strings.ToLower(getField(record, "success")) != "false"
		isTransaction := isTransactionSample(label, url, responseMessage)
		isRequest := isRequestSample(url)

		if isRequest {
			totalRT += elapsed
			if minTs == 0 || ts < minTs {
				minTs = ts
			}
			if endTs := ts + elapsed; endTs > maxEndTs {
				maxEndTs = endTs
			}
			totalRequests++
			bucket.Count++
			bucket.TotalRTMs += elapsed
			// 收集响应时间用于百分位数计算
			bucket.ElapsedMs = append(bucket.ElapsedMs, float64(elapsed))
			// 收集字节数（如果 JTL 中有 bytes 字段）
			if bytesStr := getField(record, "bytes"); bytesStr != "" {
				if bytesVal, parseErr := strconv.ParseInt(bytesStr, 10, 64); parseErr == nil {
					bucket.TotalBytes += bytesVal
				}
			}
			if success {
				successRequests++
				bucket.Success++
			} else {
				errorRequests++
				bucket.Error++
			}
		}

		if isTransaction {
			totalTransactionRT += elapsed
			if minTs == 0 || ts < minTs {
				minTs = ts
			}
			if endTs := ts + elapsed; endTs > maxEndTs {
				maxEndTs = endTs
			}
			totalTransactions++
			if success {
				successTransactions++
				bucket.TransactionSuccess++
			} else {
				errorTransactions++
				bucket.TransactionError++
			}
			bucket.TransactionCount++
			bucket.TransactionRTMs += elapsed
		}

		concurrencyValue := 0
		if raw := getField(record, "allThreads"); raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
				concurrencyValue = parsed
			}
		} else if raw := getField(record, "grpThreads"); raw != "" {
			if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
				concurrencyValue = parsed
			}
		}
		if concurrencyValue > bucket.MaxConcurrency {
			bucket.MaxConcurrency = concurrencyValue
		}
	}

	for _, currentPath := range resultPaths {
		file, err := os.Open(currentPath)
		if err != nil {
			continue
		}
		reader := csv.NewReader(file)
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1

		headers, err := reader.Read()
		if err != nil {
			file.Close()
			continue
		}
		if !headersLoaded {
			for i, h := range headers {
				colIndex[strings.TrimSpace(h)] = i
			}
			headersLoaded = true
			for _, field := range required {
				if _, ok := colIndex[field]; !ok {
					file.Close()
					empty := &LiveExecutionMetrics{Status: execution.Status, Points: []LiveMetricPoint{}}
					setCachedLiveMetrics(id, cacheSignature, empty)
					return empty, nil
				}
			}
		}

		for {
			record, err := reader.Read()
			if err != nil {
				break
			}
			processRecord(record)
		}
		file.Close()
	}

	sort.Slice(bucketOrder, func(i, j int) bool { return bucketOrder[i] < bucketOrder[j] })

	points := make([]LiveMetricPoint, 0, len(bucketOrder))
	cumulativeRequests := 0
	peakTPS := 0.0
	currentTPS := 0.0
	currentRT := 0.0
	peakRequestRate := 0.0
	currentRequestRate := 0.0
	peakConcurrency := 0
	currentConcurrency := 0

	for _, sec := range bucketOrder {
		bucket := buckets[sec]
		cumulativeRequests += bucket.Count
		avgRT := 0.0
		if totalRequests > 0 {
			if bucket.Count > 0 {
				avgRT = float64(bucket.TotalRTMs) / float64(bucket.Count)
			}
		} else if bucket.TransactionCount > 0 {
			avgRT = float64(bucket.TransactionRTMs) / float64(bucket.TransactionCount)
		}
		errorRate := 0.0
		successRate := 0.0
		if totalRequests > 0 {
			if bucket.Count > 0 {
				errorRate = float64(bucket.Error) * 100 / float64(bucket.Count)
				successRate = float64(bucket.Success) * 100 / float64(bucket.Count)
			}
		} else if bucket.TransactionCount > 0 {
			errorRate = float64(bucket.TransactionError) * 100 / float64(bucket.TransactionCount)
			successRate = float64(bucket.TransactionSuccess) * 100 / float64(bucket.TransactionCount)
		}
		tps := float64(bucket.TransactionCount)
		if tps <= 0 {
			tps = float64(bucket.Count)
		}
		requestRate := float64(bucket.Count)
		if tps > peakTPS {
			peakTPS = tps
		}
		if requestRate > peakRequestRate {
			peakRequestRate = requestRate
		}
		if bucket.MaxConcurrency > peakConcurrency {
			peakConcurrency = bucket.MaxConcurrency
		}
		currentTPS = tps
		currentRequestRate = requestRate
		currentRT = avgRT
		currentConcurrency = bucket.MaxConcurrency

		// 计算 P95/P99 百分位数
		p95RT := calculatePercentile(bucket.ElapsedMs, 95)
		p99RT := calculatePercentile(bucket.ElapsedMs, 99)

		// 计算每秒字节数（时间窗口为1秒，所以直接取 TotalBytes）
		bytesPerSec := float64(bucket.TotalBytes)

		// 获取当前时间窗口的错误数
		errorCount := bucket.Error
		if totalRequests <= 0 && bucket.TransactionCount > 0 {
			errorCount = bucket.TransactionError
		}

		points = append(points, LiveMetricPoint{
			Timestamp:     time.Unix(sec, 0).Format("15:04:05"),
			EpochSecond:   sec,
			TPS:           tps,
			RequestRate:   requestRate,
			AvgRT:         avgRT,
			SuccessRate:   successRate,
			ErrorRate:     errorRate,
			Concurrency:   bucket.MaxConcurrency,
			TotalRequests: cumulativeRequests,
			P95RT:         p95RT,
			P99RT:         p99RT,
			ErrorCount:    errorCount,
			BytesPerSec:   bytesPerSec,
		})
	}

	durationSeconds := int64(0)
	if minTs > 0 && maxEndTs >= minTs {
		durationSeconds = (maxEndTs - minTs) / 1000
	}
	avgTPS := 0.0
	avgRequestRate := 0.0
	if durationSeconds > 0 {
		baseTPSCount := totalTransactions
		if baseTPSCount <= 0 {
			baseTPSCount = totalRequests
		}
		avgTPS = float64(baseTPSCount) / float64(durationSeconds)
		avgRequestRate = float64(totalRequests) / float64(durationSeconds)
	}
	avgRT := 0.0
	if totalRequests > 0 {
		avgRT = float64(totalRT) / float64(totalRequests)
	} else if totalTransactions > 0 {
		avgRT = float64(totalTransactionRT) / float64(totalTransactions)
	}
	successRate := 0.0
	errorRate := 0.0
	if totalRequests > 0 {
		successRate = float64(successRequests) * 100 / float64(totalRequests)
		errorRate = float64(errorRequests) * 100 / float64(totalRequests)
	} else if totalTransactions > 0 {
		successRate = float64(successTransactions) * 100 / float64(totalTransactions)
		errorRate = float64(errorTransactions) * 100 / float64(totalTransactions)
	}

	hasTransactionSamples := totalTransactions > 0
	primaryThroughputLabel := "请求次数（次/秒）"
	primaryThroughputField := "request_rate"
	primaryThroughputUnit := "req/s"
	currentPrimaryThroughput := currentRequestRate
	avgPrimaryThroughput := avgRequestRate
	peakPrimaryThroughput := peakRequestRate
	if hasTransactionSamples {
		primaryThroughputLabel = "TPS（事务/s）"
		primaryThroughputField = "tps"
		primaryThroughputUnit = "tps"
		currentPrimaryThroughput = currentTPS
		avgPrimaryThroughput = avgTPS
		peakPrimaryThroughput = peakTPS
	}

	metrics := &LiveExecutionMetrics{
		Status:                   execution.Status,
		TotalRequests:            totalRequests,
		SuccessRequests:          successRequests,
		ErrorRequests:            errorRequests,
		TotalTransactions:        totalTransactions,
		SuccessTransactions:      successTransactions,
		ErrorTransactions:        errorTransactions,
		HasTransactionSamples:    hasTransactionSamples,
		PrimaryThroughputLabel:   primaryThroughputLabel,
		PrimaryThroughputField:   primaryThroughputField,
		PrimaryThroughputUnit:    primaryThroughputUnit,
		CurrentPrimaryThroughput: currentPrimaryThroughput,
		AvgPrimaryThroughput:     avgPrimaryThroughput,
		PeakPrimaryThroughput:    peakPrimaryThroughput,
		CurrentTPS:               currentTPS,
		AvgTPS:                   avgTPS,
		PeakTPS:                  peakTPS,
		CurrentRequestRate:       currentRequestRate,
		AvgRequestRate:           avgRequestRate,
		PeakRequestRate:          peakRequestRate,
		CurrentRT:                currentRT,
		AvgRT:                    avgRT,
		SuccessRate:              successRate,
		ErrorRate:                errorRate,
		CurrentConcurrency:       currentConcurrency,
		PeakConcurrency:          peakConcurrency,
		DurationSeconds:          durationSeconds,
		Points:                   points,
	}
	setCachedLiveMetrics(id, cacheSignature, metrics)
	return metrics, nil
}

func buildLiveMetricsSignature(status string, resultPaths []string) string {
	var builder strings.Builder
	builder.WriteString(status)
	for _, currentPath := range resultPaths {
		builder.WriteString("|")
		builder.WriteString(currentPath)
		if info, err := os.Stat(currentPath); err == nil {
			builder.WriteString(fmt.Sprintf(":%d:%d", info.Size(), info.ModTime().UnixNano()))
			continue
		}
		builder.WriteString(":missing")
	}
	return builder.String()
}

func getCachedLiveMetrics(execID int64, signature string) *LiveExecutionMetrics {
	liveMetricsCacheMu.RLock()
	entry, ok := liveMetricsCache[execID]
	liveMetricsCacheMu.RUnlock()
	if !ok || entry.Signature != signature || time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry.Metrics
}

func setCachedLiveMetrics(execID int64, signature string, metrics *LiveExecutionMetrics) {
	liveMetricsCacheMu.Lock()
	liveMetricsCache[execID] = liveMetricsCacheEntry{
		Signature: signature,
		ExpiresAt: time.Now().Add(liveMetricsCacheTTL),
		Metrics:   metrics,
	}
	liveMetricsCacheMu.Unlock()
}

// StopExecution 停止执行
func StopExecution(id int64) error {
	// 从进程 map 中获取 cmd
	value, ok := executionProcesses.Load(id)
	if !ok {
		return fmt.Errorf("执行不在运行中或已结束")
	}

	processGroup, ok := value.(*executionProcessGroup)
	if !ok {
		return fmt.Errorf("进程信息无效")
	}

	// 调用 cancel 取消 context（用于超时机制）
	if processGroup.Cancel != nil {
		processGroup.Cancel()
	}

	// 杀死进程组（优先）或单个进程
	for _, cmd := range processGroup.Commands {
		if cmd != nil && cmd.Process != nil {
			// 先尝试杀死整个进程组
			if !killProcessGroup(cmd) {
				// 回退到只杀主进程
				if err := cmd.Process.Kill(); err != nil {
					return fmt.Errorf("停止进程失败: %w", err)
				}
			}
		}
	}

	// 更新状态为 stopped
	endTime := time.Now().Format("2006-01-02 15:04:05")

	// 查询开始时间，计算执行时长
	var startTimeStr string
	dbErr := database.DB.QueryRow("SELECT start_time FROM executions WHERE id = ?", id).Scan(&startTimeStr)
	var duration int64
	if dbErr == nil {
		startTime, parseErr := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if parseErr == nil {
			duration = int64(time.Since(startTime).Seconds())
		}
	}

	_, err := database.DB.Exec(
		"UPDATE executions SET status = ?, end_time = ?, duration = ? WHERE id = ?",
		"stopped", endTime, duration, id,
	)
	if err != nil {
		return fmt.Errorf("更新执行状态失败: %w", err)
	}

	return nil
}

// GetExecutionLogPath 获取执行日志路径
func GetExecutionLogPath(id int64) (string, error) {
	var logPath string
	err := database.DB.QueryRow(
		"SELECT log_path FROM executions WHERE id = ?",
		id,
	).Scan(&logPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("执行记录不存在")
		}
		return "", fmt.Errorf("查询执行记录失败: %w", err)
	}
	return logPath, nil
}

// GetExecutionLog 获取执行日志内容
func GetExecutionLog(id int64) (string, error) {
	// 先获取执行记录的日志路径
	var logPath string
	err := database.DB.QueryRow(
		"SELECT log_path FROM executions WHERE id = ?",
		id,
	).Scan(&logPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("执行记录不存在")
		}
		return "", fmt.Errorf("查询执行记录失败: %w", err)
	}

	if logPath == "" {
		return "", fmt.Errorf("日志路径为空")
	}

	// 读取日志文件
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("日志文件不存在")
		}
		return "", fmt.Errorf("读取日志文件失败: %w", err)
	}

	return string(content), nil
}

// cleanOrphanedJMeterProcesses 清理残留的 JMeter 进程（服务启动时调用）
// 仅在 Linux/macOS 下执行
func cleanOrphanedJMeterProcesses() {
	if runtime.GOOS == "windows" {
		return
	}

	// 使用 pgrep 查找残留的 JMeter 进程
	output, err := exec.Command("pgrep", "-f", "ApacheJMeter").Output()
	if err != nil {
		// pgrep 返回非零退出码表示没有找到进程，这是正常情况
		return
	}

	pids := strings.TrimSpace(string(output))
	if pids == "" {
		return
	}

	fmt.Printf("发现残留的 JMeter 进程，正在清理...\n")

	// 使用 pkill 强制清理残留的 JMeter 进程
	if err := exec.Command("pkill", "-9", "-f", "ApacheJMeter").Run(); err != nil {
		fmt.Printf("清理残留 JMeter 进程失败: %v\n", err)
	} else {
		fmt.Printf("已清理残留的 JMeter 进程\n")
	}
}

// CleanupStaleExecutions 清理陈旧的执行记录（服务启动时调用）
// 将所有 status='running' 的记录更新为 'failed'
func CleanupStaleExecutions() error {
	result, err := database.DB.Exec(
		"UPDATE executions SET status = ?, remarks = ? WHERE status = ?",
		"failed", "服务重启，执行中断", "running",
	)
	if err != nil {
		return fmt.Errorf("清理陈旧执行记录失败: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		fmt.Printf("已清理 %d 条未完成的执行记录\n", affected)
	}

	// 清理残留的 JMeter 进程
	cleanOrphanedJMeterProcesses()

	return nil
}

// parseJTLResults 解析 JTL 结果文件
func parseJTLResults(jtlPaths ...string) (string, error) {
	colIndex := map[string]int{}
	headersLoaded := false
	type recordData []string
	records := make([]recordData, 0)

	for _, jtlPath := range jtlPaths {
		if strings.TrimSpace(jtlPath) == "" {
			continue
		}
		file, err := os.Open(jtlPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("打开 JTL 文件失败: %w", err)
		}

		reader := csv.NewReader(file)
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1

		headers, err := reader.Read()
		if err != nil {
			file.Close()
			if err == io.EOF {
				continue
			}
			return "", fmt.Errorf("读取 JTL 表头失败: %w", err)
		}

		if !headersLoaded {
			for i, h := range headers {
				colIndex[strings.TrimSpace(h)] = i
			}
			headersLoaded = true
		}

		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			records = append(records, record)
		}
		file.Close()
	}

	if !headersLoaded {
		return "{}", nil
	}

	// 需要的列
	elapsedIdx, hasElapsed := colIndex["elapsed"]
	successIdx, hasSuccess := colIndex["success"]
	labelIdx, hasLabel := colIndex["label"]
	timeStampIdx, hasTimeStamp := colIndex["timeStamp"]
	bytesIdx, hasBytes := colIndex["bytes"]             // 接收字节数
	sentBytesIdx, hasSentBytes := colIndex["sentBytes"] // 发送字节数
	responseMsgIdx, hasResponseMsg := colIndex["responseMessage"]
	urlIdx, hasURL := colIndex["URL"]

	if !hasElapsed || !hasSuccess {
		return "", fmt.Errorf("JTL 文件缺少必要的列")
	}

	// 统计数据
	totalSamples := 0
	totalElapsed := int64(0)
	minElapsed := int64(-1)
	maxElapsed := int64(0)
	errorCount := 0
	successCount := 0
	totalBytes := int64(0)     // 总接收字节数
	totalSentBytes := int64(0) // 总发送字节数
	var elapsedList []float64
	requestSamples := 0
	transactionSamples := 0
	requestSuccessCount := 0
	requestErrorCount := 0
	transactionSuccessCount := 0
	transactionErrorCount := 0
	minTimeStamp := int64(0)
	maxEndTimeStamp := int64(0)
	type samplerAggregate struct {
		Label       string
		URL         string
		Count       int
		Success     int
		Error       int
		TotalRT     int64
		MinRT       int64
		MaxRT       int64
		ElapsedList []float64
	}
	samplerStats := make(map[string]*samplerAggregate)

	// 逐行读取数据
	for _, record := range records {
		label := ""
		if hasLabel && labelIdx < len(record) {
			label = record[labelIdx]
		}
		responseMessage := ""
		if hasResponseMsg && responseMsgIdx < len(record) {
			responseMessage = record[responseMsgIdx]
		}
		url := ""
		if hasURL && urlIdx < len(record) {
			url = record[urlIdx]
		}
		success := successIdx < len(record) && strings.ToLower(record[successIdx]) != "false"
		transactionSample := isTransactionSample(label, url, responseMessage)
		requestSample := isRequestSample(url)

		if elapsedIdx < len(record) && requestSample {
			elapsed, err := strconv.ParseInt(record[elapsedIdx], 10, 64)
			if err == nil {
				totalSamples++
				requestSamples++
				totalElapsed += elapsed
				elapsedList = append(elapsedList, float64(elapsed))

				if minElapsed == -1 || elapsed < minElapsed {
					minElapsed = elapsed
				}
				if elapsed > maxElapsed {
					maxElapsed = elapsed
				}

				if hasTimeStamp && timeStampIdx < len(record) {
					if ts, err := strconv.ParseInt(record[timeStampIdx], 10, 64); err == nil {
						if minTimeStamp == 0 || ts < minTimeStamp {
							minTimeStamp = ts
						}
						endTs := ts + elapsed
						if endTs > maxEndTimeStamp {
							maxEndTimeStamp = endTs
						}
					}
				}

				samplerKey := strings.TrimSpace(label) + "|" + strings.TrimSpace(url)
				sampler := samplerStats[samplerKey]
				if sampler == nil {
					sampler = &samplerAggregate{
						Label: label,
						URL:   url,
						MinRT: -1,
					}
					samplerStats[samplerKey] = sampler
				}
				sampler.Count++
				sampler.TotalRT += elapsed
				sampler.ElapsedList = append(sampler.ElapsedList, float64(elapsed))
				if sampler.MinRT == -1 || elapsed < sampler.MinRT {
					sampler.MinRT = elapsed
				}
				if elapsed > sampler.MaxRT {
					sampler.MaxRT = elapsed
				}
			}
		}

		if requestSample {
			if success {
				successCount++
				requestSuccessCount++
				if sampler := samplerStats[strings.TrimSpace(label)+"|"+strings.TrimSpace(url)]; sampler != nil {
					sampler.Success++
				}
			} else {
				errorCount++
				requestErrorCount++
				if sampler := samplerStats[strings.TrimSpace(label)+"|"+strings.TrimSpace(url)]; sampler != nil {
					sampler.Error++
				}
			}
		}

		if transactionSample {
			transactionSamples++
			if success {
				transactionSuccessCount++
			} else {
				transactionErrorCount++
			}
		}

		// 统计接收字节数
		if requestSample && hasBytes && bytesIdx < len(record) {
			bytes, err := strconv.ParseInt(record[bytesIdx], 10, 64)
			if err == nil {
				totalBytes += bytes
			}
		}

		// 统计发送字节数
		if requestSample && hasSentBytes && sentBytesIdx < len(record) {
			sentBytes, err := strconv.ParseInt(record[sentBytesIdx], 10, 64)
			if err == nil {
				totalSentBytes += sentBytes
			}
		}
	}

	if totalSamples == 0 {
		return "{}", nil
	}

	// 计算平均值
	avgElapsed := float64(totalElapsed) / float64(totalSamples)

	// 计算百分位数
	p50, p90, p95, p99 := calculatePercentiles(elapsedList)

	// 基于首个请求开始时间到最后一个请求结束时间计算吞吐量
	durationMs := maxEndTimeStamp - minTimeStamp
	if durationMs < 0 {
		durationMs = 0
	}
	durationSeconds := float64(durationMs) / 1000.0
	throughput := 0.0
	requestRate := 0.0
	transactionTPS := 0.0
	baseTransactionSamples := transactionSamples
	if baseTransactionSamples <= 0 {
		baseTransactionSamples = requestSamples
	}
	if durationSeconds > 0 {
		throughput = float64(requestSamples) / durationSeconds
		requestRate = float64(requestSamples) / durationSeconds
		transactionTPS = float64(baseTransactionSamples) / durationSeconds
	} else {
		throughput = float64(requestSamples)
		requestRate = float64(requestSamples)
		transactionTPS = float64(baseTransactionSamples)
	}
	primaryThroughput := requestRate
	primaryThroughputLabel := "请求次数（次/秒）"
	primaryThroughputField := "request_rate"
	primaryThroughputUnit := "req/s"
	sampleBasis := "request"
	if transactionSamples > 0 {
		primaryThroughput = transactionTPS
		primaryThroughputLabel = "TPS（事务/s）"
		primaryThroughputField = "transaction_tps"
		primaryThroughputUnit = "tps"
		sampleBasis = "transaction"
	}
	errorRate := 0.0
	successRate := 0.0
	if requestSamples > 0 {
		errorRate = float64(requestErrorCount) * 100.0 / float64(requestSamples)
		successRate = float64(requestSuccessCount) * 100.0 / float64(requestSamples)
	}
	receivedBytesPerSec := 0.0
	sentBytesPerSec := 0.0
	if durationSeconds > 0 {
		receivedBytesPerSec = float64(totalBytes) / durationSeconds
		sentBytesPerSec = float64(totalSentBytes) / durationSeconds
	}

	if minElapsed == -1 {
		minElapsed = 0
	}

	samplerList := make([]map[string]interface{}, 0, len(samplerStats))
	for _, sampler := range samplerStats {
		avgRT := 0.0
		errorRatePerSampler := 0.0
		successRatePerSampler := 0.0
		throughputPerSampler := 0.0
		if sampler.Count > 0 {
			avgRT = float64(sampler.TotalRT) / float64(sampler.Count)
			errorRatePerSampler = float64(sampler.Error) * 100 / float64(sampler.Count)
			successRatePerSampler = float64(sampler.Success) * 100 / float64(sampler.Count)
		}
		if durationSeconds > 0 {
			throughputPerSampler = float64(sampler.Count) / durationSeconds
		} else {
			throughputPerSampler = float64(sampler.Count)
		}
		p50Sampler, p90Sampler, p95Sampler, p99Sampler := calculatePercentiles(sampler.ElapsedList)
		if sampler.MinRT == -1 {
			sampler.MinRT = 0
		}
		samplerList = append(samplerList, map[string]interface{}{
			"label":        sampler.Label,
			"url":          sampler.URL,
			"count":        sampler.Count,
			"success":      sampler.Success,
			"error":        sampler.Error,
			"error_rate":   errorRatePerSampler,
			"success_rate": successRatePerSampler,
			"avg_rt":       avgRT,
			"min_rt":       sampler.MinRT,
			"max_rt":       sampler.MaxRT,
			"p50":          p50Sampler,
			"p90":          p90Sampler,
			"p95":          p95Sampler,
			"p99":          p99Sampler,
			"throughput":   throughputPerSampler,
		})
	}
	sort.Slice(samplerList, func(i, j int) bool {
		leftCount := intValueFromMap(samplerList[i], "count")
		rightCount := intValueFromMap(samplerList[j], "count")
		if leftCount == rightCount {
			leftError := intValueFromMap(samplerList[i], "error")
			rightError := intValueFromMap(samplerList[j], "error")
			return leftError > rightError
		}
		return leftCount > rightCount
	})
	if len(samplerList) > 30 {
		samplerList = samplerList[:30]
	}

	conclusion := buildExecutionConclusion(totalSamples, requestSamples, transactionSamples, requestErrorCount, errorRate, successRate, avgElapsed, p95, p99, primaryThroughputLabel, primaryThroughput, samplerList)

	// 构建结果
	result := map[string]interface{}{
		"total_samples":               totalSamples,
		"success_samples":             successCount,
		"error_samples":               errorCount,
		"request_success_samples":     requestSuccessCount,
		"request_error_samples":       requestErrorCount,
		"transaction_success_samples": transactionSuccessCount,
		"transaction_error_samples":   transactionErrorCount,
		"avg_response_time":           avgElapsed,
		"min_response_time":           minElapsed,
		"max_response_time":           maxElapsed,
		"throughput":                  throughput,
		"transaction_tps":             transactionTPS,
		"request_rate":                requestRate,
		"primary_throughput":          primaryThroughput,
		"primary_throughput_label":    primaryThroughputLabel,
		"primary_throughput_field":    primaryThroughputField,
		"primary_throughput_unit":     primaryThroughputUnit,
		"sample_basis":                sampleBasis,
		"error_rate":                  errorRate,
		"success_rate":                successRate,
		"transaction_samples":         transactionSamples,
		"request_samples":             requestSamples,
		"p50":                         p50,
		"p90":                         p90,
		"p95":                         p95,
		"p99":                         p99,
		"sample_span_ms":              durationMs,
		"received_bytes":              totalBytes,
		"sent_bytes":                  totalSentBytes,
		"received_bytes_per_sec":      receivedBytesPerSec,
		"sent_bytes_per_sec":          sentBytesPerSec,
		"sampler_stats":               samplerList,
		"conclusion":                  conclusion,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// calculatePercentiles 计算百分位数
func calculatePercentiles(values []float64) (p50, p90, p95, p99 float64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}

	// 排序
	sort.Float64s(values)

	n := float64(len(values))

	// P50
	idx50 := int(0.5 * (n - 1))
	if idx50 >= len(values) {
		idx50 = len(values) - 1
	}
	p50 = values[idx50]

	// P90
	idx90 := int(0.9 * (n - 1))
	if idx90 >= len(values) {
		idx90 = len(values) - 1
	}
	p90 = values[idx90]

	// P95
	idx95 := int(0.95 * (n - 1))
	if idx95 >= len(values) {
		idx95 = len(values) - 1
	}
	p95 = values[idx95]

	// P99
	idx99 := int(0.99 * (n - 1))
	if idx99 >= len(values) {
		idx99 = len(values) - 1
	}
	p99 = values[idx99]

	return p50, p90, p95, p99
}

func intValueFromMap(item map[string]interface{}, key string) int {
	value, ok := item[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func floatValueFromMap(item map[string]interface{}, key string) float64 {
	value, ok := item[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func stringValueFromMap(item map[string]interface{}, key string) string {
	value, ok := item[key]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func buildExecutionConclusion(totalSamples, requestSamples, transactionSamples, requestErrorCount int, errorRate, successRate, avgRT, p95, p99 float64, throughputLabel string, throughput float64, samplerList []map[string]interface{}) map[string]interface{} {
	level := "success"
	title := "压测结果整体稳定"
	summary := "本次执行的核心指标整体平稳，可继续结合接口排行和错误分析做针对性复盘。"
	if totalSamples == 0 {
		level = "warning"
		title = "未采集到有效请求样本"
		summary = "执行已结束，但结果中没有有效请求样本，请优先检查脚本结构、监听器和结果文件配置。"
	} else if requestSamples > 0 && requestErrorCount == requestSamples {
		level = "danger"
		title = "本次压测请求全部失败"
		summary = "所有请求样本均失败，当前结果更像环境、鉴权、依赖或脚本断言问题，而不是正常的性能压测结果。"
	} else if errorRate >= 20 {
		level = "danger"
		title = "错误率偏高，结果不达标"
		summary = "本次压测已出现明显失败流量，建议优先定位失败接口与失败时间段，再评估性能指标是否还有参考价值。"
	} else if errorRate >= 5 {
		level = "warning"
		title = "存在失败流量，需要重点复盘"
		summary = "本次压测不是全绿结果，建议优先查看错误分析与最慢接口，确认失败是否集中在特定接口或特定时间窗口。"
	} else if p95 > avgRT*2 && avgRT > 0 {
		level = "warning"
		title = "响应抖动较大"
		summary = "虽然整体成功率较高，但长尾延迟明显高于平均水平，建议重点关注 P95/P99 与慢接口排行。"
	}

	highlights := make([]string, 0, 4)
	highlights = append(highlights, fmt.Sprintf("成功率 %.2f%% / 错误率 %.2f%%", successRate, errorRate))
	highlights = append(highlights, fmt.Sprintf("%s %.2f", throughputLabel, throughput))
	highlights = append(highlights, fmt.Sprintf("平均 RT %.2f ms / P95 %.2f ms / P99 %.2f ms", avgRT, p95, p99))
	if transactionSamples > 0 {
		highlights = append(highlights, fmt.Sprintf("共识别事务样本 %d，请优先用 TPS（事务/s）口径复盘。", transactionSamples))
	}

	recommendations := make([]string, 0, 4)
	switch level {
	case "danger":
		recommendations = append(recommendations,
			"先看错误分析，确认失败是断言失败、业务失败还是网络/依赖问题。",
			"优先检查 Top 错误接口和错误时间线，判断失败是否集中在某一批接口或某一时间段。",
		)
	case "warning":
		recommendations = append(recommendations,
			"结合慢接口排行与实时趋势，确认 RT 抬升是否与并发提升同步发生。",
			"将本次结果与基线执行对比，判断是回归退化还是单次波动。",
		)
	default:
		recommendations = append(recommendations,
			"建议保留当前结果作为基线，后续执行可直接做回归对比。",
			"继续关注 Top 接口的长尾 RT，避免平均值掩盖局部瓶颈。",
		)
	}

	if len(samplerList) > 0 {
		hottest := samplerList[0]
		highlights = append(highlights, fmt.Sprintf("样本最多的接口是 %s（%d 次）", stringValueFromMap(hottest, "label"), intValueFromMap(hottest, "count")))
		slowest := samplerList[0]
		riskiest := samplerList[0]
		for _, item := range samplerList[1:] {
			if floatValueFromMap(item, "avg_rt") > floatValueFromMap(slowest, "avg_rt") {
				slowest = item
			}
			if intValueFromMap(item, "error") > intValueFromMap(riskiest, "error") {
				riskiest = item
			}
		}
		recommendations = append(recommendations,
			fmt.Sprintf("优先关注最慢接口 %s（平均 %.2f ms）。", stringValueFromMap(slowest, "label"), floatValueFromMap(slowest, "avg_rt")),
			fmt.Sprintf("若要快速定位风险，先检查错误最多的接口 %s。", stringValueFromMap(riskiest, "label")),
		)
	}

	return map[string]interface{}{
		"level":           level,
		"title":           title,
		"summary":         summary,
		"highlights":      dedupeStrings(highlights),
		"recommendations": dedupeStrings(recommendations),
	}
}

func mergeJTLFiles(inputPaths []string, outputPath string) error {
	log.Printf("[JTL合并] 输入文件: %v", inputPaths)

	type filePayload struct {
		headers []string
		records [][]string
	}

	var payloads []filePayload
	for _, inputPath := range inputPaths {
		if strings.TrimSpace(inputPath) == "" {
			continue
		}
		file, err := os.Open(inputPath)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("[JTL合并][警告] 结果文件不存在: %s", inputPath)
				continue
			}
			return fmt.Errorf("打开结果文件失败 %s: %w", inputPath, err)
		}

		reader := csv.NewReader(file)
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1
		headers, err := reader.Read()
		if err != nil {
			file.Close()
			if err == io.EOF {
				log.Printf("[JTL合并][警告] 结果文件为空: %s", inputPath)
				continue
			}
			return fmt.Errorf("读取结果文件表头失败 %s: %w", inputPath, err)
		}
		var records [][]string
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			records = append(records, record)
		}
		file.Close()
		payloads = append(payloads, filePayload{headers: headers, records: records})
	}

	if len(payloads) == 0 {
		return fmt.Errorf("没有可合并的结果文件")
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("创建合并结果目录失败: %w", err)
	}
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建合并结果文件失败: %w", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	if err := writer.Write(payloads[0].headers); err != nil {
		return fmt.Errorf("写入合并表头失败: %w", err)
	}
	for _, payload := range payloads {
		for _, record := range payload.records {
			if err := writer.Write(record); err != nil {
				return fmt.Errorf("写入合并记录失败: %w", err)
			}
		}
	}
	writer.Flush()
	totalRecords := 0
	for _, p := range payloads {
		totalRecords += len(p.records)
	}
	log.Printf("[JTL合并] 合并完成: %d 个数据源, 共 %d 条记录", len(payloads), totalRecords)
	return writer.Error()
}

func discoverExecutionResultPaths(resultPath string) []string {
	if strings.TrimSpace(resultPath) == "" {
		return nil
	}

	paths := make([]string, 0, 3)
	if info, err := os.Stat(resultPath); err == nil && info.Size() > 0 {
		return []string{resultPath}
	}

	baseDir := filepath.Dir(resultPath)
	candidates := []string{
		filepath.Join(baseDir, "result-local.jtl"),
		filepath.Join(baseDir, "result-remote.jtl"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.Size() > 0 {
			paths = append(paths, candidate)
		}
	}
	return paths
}

func buildResponsePreview(responseData, failureMessage string) string {
	preview := strings.TrimSpace(responseData)
	if preview == "" {
		preview = strings.TrimSpace(failureMessage)
	}
	if preview == "" {
		return ""
	}
	preview = strings.ReplaceAll(preview, "\r\n", "\n")
	preview = strings.ReplaceAll(preview, "\n", " ")
	runes := []rune(preview)
	if len(runes) > 120 {
		return string(runes[:120]) + "..."
	}
	return preview
}

// DeleteExecution 删除执行记录（同时清理关联的日志和结果文件）
func DeleteExecution(id int64) error {
	// 先获取执行记录信息
	execution, err := GetExecution(id)
	if err != nil {
		return err
	}

	// 删除数据库记录
	result, err := database.DB.Exec("DELETE FROM executions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除执行记录失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("执行记录不存在")
	}

	// 清理磁盘文件（日志、JTL结果文件、报告目录）
	resultDir := filepath.Join(config.GlobalConfig.Dirs.Results, fmt.Sprintf("%d", id))
	if err := os.RemoveAll(resultDir); err != nil {
		// 记录错误但不中断
		fmt.Printf("删除执行结果目录失败 %s: %v\n", resultDir, err)
	}

	// 单独清理可能残留的文件
	if execution.LogPath != "" {
		if err := os.Remove(execution.LogPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("删除日志文件失败 %s: %v\n", execution.LogPath, err)
		}
	}
	if execution.ResultPath != "" {
		if err := os.Remove(execution.ResultPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("删除结果文件失败 %s: %v\n", execution.ResultPath, err)
		}
	}
	if execution.ReportPath != "" {
		if err := os.RemoveAll(execution.ReportPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("删除报告目录失败 %s: %v\n", execution.ReportPath, err)
		}
	}

	return nil
}

// ErrorRecord 单条错误记录
type ErrorRecord struct {
	Source          string `json:"source"`
	Timestamp       string `json:"timestamp"`
	Elapsed         int64  `json:"elapsed"`
	Label           string `json:"label"`
	ResponseCode    string `json:"response_code"`
	ResponseMessage string `json:"response_message"`
	ResponseData    string `json:"response_data"`
	ResponseHeaders string `json:"response_headers"`
	ThreadName      string `json:"thread_name"`
	FailureMessage  string `json:"failure_message"`
	URL             string `json:"url"`
	Bytes           int64  `json:"bytes"`
	RequestHeaders  string `json:"request_headers"` // 请求头
	RequestBody     string `json:"request_body"`    // 请求体/参数
	SentBytes       int64  `json:"sent_bytes"`      // 发送字节数
	Latency         int64  `json:"latency"`         // 延迟(ms)
	ConnectTime     int64  `json:"connect_time"`    // 连接时间(ms)
}

// ErrorType 错误类型统计
type ErrorType struct {
	Label           string        `json:"label"`
	ResponseCode    string        `json:"response_code"`
	ResponseMessage string        `json:"response_message"`
	Category        string        `json:"category"`
	ResponsePreview string        `json:"response_preview"`
	Count           int           `json:"count"`
	Percentage      float64       `json:"percentage"`
	FirstTime       string        `json:"first_time"`
	LastTime        string        `json:"last_time"`
	SampleErrors    []ErrorRecord `json:"sample_errors"` // 每种类型保留最多5条样例
}

// CodeCount 响应码分布
type CodeCount struct {
	Code       string  `json:"code"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// TimelinePoint 错误时间线点
type TimelinePoint struct {
	TimeBucket  string  `json:"time_bucket"` // "15:04:05" 格式（10秒粒度）
	ErrorCount  int     `json:"error_count"`
	SampleCount int     `json:"sample_count"`
	ErrorRate   float64 `json:"error_rate"`
}

// MessageCount Top 错误信息
type MessageCount struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type ErrorCluster struct {
	Key        string   `json:"key"`
	Label      string   `json:"label"`
	Count      int      `json:"count"`
	Percentage float64  `json:"percentage"`
	Tone       string   `json:"tone"`
	Example    string   `json:"example"`
	TopLabels  []string `json:"top_labels,omitempty"`
	TopSources []string `json:"top_sources,omitempty"`
	URL        string   `json:"url,omitempty"`
}

// ErrorAnalysis 错误分析结果
type ErrorAnalysis struct {
	TotalErrors              int             `json:"total_errors"`
	TotalSamples             int             `json:"total_samples"`
	ErrorRate                float64         `json:"error_rate"`
	ErrorTypes               []ErrorType     `json:"error_types"`
	Records                  []ErrorRecord   `json:"records"`                 // 按类型各最多10000条
	Truncated                bool            `json:"truncated"`               // 是否被截断
	TypeTruncated            map[string]bool `json:"type_truncated"`          // 哪些错误类型被截断
	DetailFieldsAvailable    bool            `json:"detail_fields_available"` // 是否记录了请求/响应详情字段
	DetailStorageHint        string          `json:"detail_storage_hint"`     // 当前结果文件的详情保存说明
	AvailableDetailFields    []string        `json:"available_detail_fields"` // 实际可用的详情列
	ExpectedDetailSources    []string        `json:"expected_detail_sources"`
	ReceivedDetailSources    []string        `json:"received_detail_sources"`
	MissingDetailSources     []string        `json:"missing_detail_sources"`
	DetailUploadWarning      string          `json:"detail_upload_warning"`
	ResponseCodeDistribution []CodeCount     `json:"response_code_distribution"`
	ErrorTimeline            []TimelinePoint `json:"error_timeline"`
	TopErrorMessages         []MessageCount  `json:"top_error_messages"`
	CategoryDistribution     []ErrorCluster  `json:"category_distribution"`
	SourceDistribution       []ErrorCluster  `json:"source_distribution"`
	APIDistribution          []ErrorCluster  `json:"api_distribution"`
	ReportHighlights         []string        `json:"report_highlights"`
}

type ErrorAnalysisOverview struct {
	TotalErrors              int             `json:"total_errors"`
	TotalSamples             int             `json:"total_samples"`
	ErrorRate                float64         `json:"error_rate"`
	ErrorTypes               []ErrorType     `json:"error_types"`
	Truncated                bool            `json:"truncated"`
	TypeTruncated            map[string]bool `json:"type_truncated"`
	DetailFieldsAvailable    bool            `json:"detail_fields_available"`
	DetailStorageHint        string          `json:"detail_storage_hint"`
	AvailableDetailFields    []string        `json:"available_detail_fields"`
	ExpectedDetailSources    []string        `json:"expected_detail_sources"`
	ReceivedDetailSources    []string        `json:"received_detail_sources"`
	MissingDetailSources     []string        `json:"missing_detail_sources"`
	DetailUploadWarning      string          `json:"detail_upload_warning"`
	ResponseCodeDistribution []CodeCount     `json:"response_code_distribution"`
	ErrorTimeline            []TimelinePoint `json:"error_timeline"`
	TopErrorMessages         []MessageCount  `json:"top_error_messages"`
	CategoryDistribution     []ErrorCluster  `json:"category_distribution"`
	SourceDistribution       []ErrorCluster  `json:"source_distribution"`
	APIDistribution          []ErrorCluster  `json:"api_distribution"`
	ReportHighlights         []string        `json:"report_highlights"`
}

type LiveMetricPoint struct {
	Timestamp     string  `json:"timestamp"`
	EpochSecond   int64   `json:"epoch_second"`
	TPS           float64 `json:"tps"`
	RequestRate   float64 `json:"request_rate"`
	AvgRT         float64 `json:"avg_rt"`
	SuccessRate   float64 `json:"success_rate"`
	ErrorRate     float64 `json:"error_rate"`
	Concurrency   int     `json:"concurrency"`
	TotalRequests int     `json:"total_requests"`
	P95RT         float64 `json:"p95_rt"`        // P95 响应时间
	P99RT         float64 `json:"p99_rt"`        // P99 响应时间
	ErrorCount    int     `json:"error_count"`   // 当前时间窗口内的错误数
	BytesPerSec   float64 `json:"bytes_per_sec"` // 每秒传输字节数
}

type LiveExecutionMetrics struct {
	Status                   string            `json:"status"`
	TotalRequests            int               `json:"total_requests"`
	SuccessRequests          int               `json:"success_requests"`
	ErrorRequests            int               `json:"error_requests"`
	TotalTransactions        int               `json:"total_transactions"`
	SuccessTransactions      int               `json:"success_transactions"`
	ErrorTransactions        int               `json:"error_transactions"`
	HasTransactionSamples    bool              `json:"has_transaction_samples"`
	PrimaryThroughputLabel   string            `json:"primary_throughput_label"`
	PrimaryThroughputField   string            `json:"primary_throughput_field"`
	PrimaryThroughputUnit    string            `json:"primary_throughput_unit"`
	CurrentPrimaryThroughput float64           `json:"current_primary_throughput"`
	AvgPrimaryThroughput     float64           `json:"avg_primary_throughput"`
	PeakPrimaryThroughput    float64           `json:"peak_primary_throughput"`
	CurrentTPS               float64           `json:"current_tps"`
	AvgTPS                   float64           `json:"avg_tps"`
	PeakTPS                  float64           `json:"peak_tps"`
	CurrentRequestRate       float64           `json:"current_request_rate"`
	AvgRequestRate           float64           `json:"avg_request_rate"`
	PeakRequestRate          float64           `json:"peak_request_rate"`
	CurrentRT                float64           `json:"current_rt"`
	AvgRT                    float64           `json:"avg_rt"`
	SuccessRate              float64           `json:"success_rate"`
	ErrorRate                float64           `json:"error_rate"`
	CurrentConcurrency       int               `json:"current_concurrency"`
	PeakConcurrency          int               `json:"peak_concurrency"`
	DurationSeconds          int64             `json:"duration_seconds"`
	Points                   []LiveMetricPoint `json:"points"`
}

type liveBucket struct {
	Count              int
	Success            int
	Error              int
	TotalRTMs          int64
	MaxConcurrency     int
	TransactionCount   int
	TransactionSuccess int
	TransactionError   int
	TransactionRTMs    int64
	ElapsedMs          []float64 // 存储该时间窗口内所有请求的响应时间，用于计算百分位数
	TotalBytes         int64     // 该时间窗口内的总字节数
}

// calculatePercentile 计算百分位数（使用简单排序法）
func calculatePercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	// 复制切片以避免修改原始数据
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	index := percentile / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	// 线性插值
	fraction := index - float64(lower)
	return sorted[lower] + fraction*(sorted[upper]-sorted[lower])
}

type errorDetailEntry struct {
	Source          string `json:"source"`
	Timestamp       string `json:"timestamp"`
	Label           string `json:"label"`
	ThreadName      string `json:"thread_name"`
	Elapsed         int64  `json:"elapsed"`
	URL             string `json:"url"`
	ResponseCode    string `json:"response_code"`
	ResponseMessage string `json:"response_message"`
	FailureMessage  string `json:"failure_message"`
	RequestHeaders  string `json:"request_headers"`
	RequestBody     string `json:"request_body"`
	ResponseHeaders string `json:"response_headers"`
	ResponseData    string `json:"response_data"`
}

func xmlEscape(value string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(value))
	return buf.String()
}

func buildErrorDetailListenerXML() string {
	script := `import groovy.json.JsonOutput
import java.net.InetAddress
import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.nio.file.StandardOpenOption

if (!prev.isSuccessful()) {
    try {
        def outputFile = props.get("jmeterAdmin.errorDetailFile")
        if (!outputFile) {
            log.warn("错误明细监听器未配置输出文件，跳过写入")
            return
        }

        def safeCall = { target, methodName, defaultValue = "" ->
            if (target == null) {
                return defaultValue
            }
            try {
                if (target.metaClass.respondsTo(target, methodName)) {
                    def value = target."${methodName}"()
                    return value == null ? defaultValue : value
                }
            } catch (Exception ignored) {}
            return defaultValue
        }

        def extractFailureMessage = {
            def messages = []
            try {
                def assertionResults = safeCall(prev, "getAssertionResults", null)
                if (assertionResults != null) {
                    assertionResults.each { result ->
                        if (result == null) {
                            return
                        }
                        try {
                            def failure = safeCall(result, "isFailure", false)
                            def error = safeCall(result, "isError", false)
                            if (failure || error) {
                                def msg = safeCall(result, "getFailureMessage", "")
                                if (msg) {
                                    messages << String.valueOf(msg)
                                }
                            }
                        } catch (Exception ignored) {}
                    }
                }
            } catch (Exception ignored) {}

            if (!messages.isEmpty()) {
                return messages.join(" | ")
            }

            def responseMessage = safeCall(prev, "getResponseMessage", "")
            if (responseMessage) {
                return String.valueOf(responseMessage)
            }

            return "Sample failed"
        }

        def source = props.get("jmeterAdmin.detailSource")
        if (!source) {
            try {
                source = InetAddress.getLocalHost().getHostAddress() + " (" + InetAddress.getLocalHost().getHostName() + ")"
            } catch (Exception ignored) {
                source = "local"
            }
        }

        def url = ""
        try {
            def rawUrl = safeCall(prev, "getURL", null)
            if (rawUrl != null) {
                url = rawUrl.toString()
            }
        } catch (Exception ignored) {}

        def payload = [
            source          : String.valueOf(source),
            timestamp       : String.valueOf(safeCall(prev, "getTimeStamp", "")),
            label           : String.valueOf(safeCall(prev, "getSampleLabel", "")),
            thread_name     : String.valueOf(safeCall(prev, "getThreadName", "")),
            elapsed         : safeCall(prev, "getTime", 0L),
            url             : url,
            response_code   : String.valueOf(safeCall(prev, "getResponseCode", "")),
            response_message: String.valueOf(safeCall(prev, "getResponseMessage", "")),
            failure_message : extractFailureMessage(),
            request_headers : String.valueOf(safeCall(prev, "getRequestHeaders", "")),
            request_body    : String.valueOf(safeCall(prev, "getSamplerData", "")),
            response_headers: String.valueOf(safeCall(prev, "getResponseHeaders", "")),
            response_data   : String.valueOf(safeCall(prev, "getResponseDataAsString", ""))
        ]

        Path target = Paths.get(outputFile)
        Files.createDirectories(target.getParent())
        Files.write(
            target,
            (JsonOutput.toJson(payload) + System.lineSeparator()).getBytes(StandardCharsets.UTF_8),
            StandardOpenOption.CREATE,
            StandardOpenOption.APPEND
        )
    } catch (Exception ex) {
        log.error("错误明细监听器写入失败", ex)
    }
}`

	return `<JSR223Listener guiclass="TestBeanGUI" testclass="JSR223Listener" testname="错误明细监听器" enabled="true">
  <stringProp name="cacheKey">true</stringProp>
  <stringProp name="filename"></stringProp>
  <stringProp name="parameters"></stringProp>
  <stringProp name="script">` + xmlEscape(script) + `</stringProp>
  <stringProp name="scriptLanguage">groovy</stringProp>
</JSR223Listener>
<hashTree/>`
}

func buildErrorDetailUploadThreadGroupXML() string {
	script := `import groovy.json.JsonOutput
import java.net.HttpURLConnection
import java.net.InetAddress
import java.net.URL
import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths

if (!"true".equalsIgnoreCase(String.valueOf(props.get("jmeterAdmin.errorDetailUploadEnabled")))) {
    return
}

def uploadUrl = props.get("jmeterAdmin.errorDetailUploadUrl")
def uploadToken = props.get("jmeterAdmin.errorDetailUploadToken")
def detailFile = props.get("jmeterAdmin.errorDetailFile")
if (!uploadUrl || !uploadToken || !detailFile) {
    return
}

Path detailPath = Paths.get(detailFile)
if (!Files.exists(detailPath) || Files.size(detailPath) == 0) {
    return
}

	def hostAddress = ""
	def hostName = ""
	try {
	    def local = InetAddress.getLocalHost()
	    hostAddress = local.getHostAddress() ?: ""
	    hostName = local.getHostName() ?: ""
	} catch (Exception ignored) {}
	def source = hostAddress
	if (hostName) {
	    source = source ? (source + " (" + hostName + ")") : hostName
	}
	if (!source) {
	    source = "unknown-slave"
	}
	def content = Files.readString(detailPath, StandardCharsets.UTF_8)
	def payload = JsonOutput.toJson([
	    token  : String.valueOf(uploadToken),
	    source : source,
	    content: content
	])

	boolean uploaded = false
	def attempt = 0
	while (!uploaded && attempt < 3) {
	    attempt++
	    HttpURLConnection connection = null
	    try {
	        connection = (HttpURLConnection) new URL(String.valueOf(uploadUrl)).openConnection()
	        connection.setConnectTimeout(10000)
	        connection.setReadTimeout(30000)
	        connection.setRequestMethod("POST")
	        connection.setDoOutput(true)
	        connection.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
	        connection.getOutputStream().withCloseable { os ->
	            os.write(payload.getBytes(StandardCharsets.UTF_8))
	        }

	        def code = connection.getResponseCode()
	        if (code >= 200 && code < 300) {
	            Files.deleteIfExists(detailPath)
	            uploaded = true
	        } else {
	            def errText = ""
	            if (connection.getErrorStream() != null) {
	                errText = connection.getErrorStream().getText("UTF-8")
	            }
	            System.err.println("Upload error details failed, attempt=" + attempt + ", code=" + code + ", body=" + errText)
	        }
	    } catch (Exception ex) {
	        System.err.println("Upload error details failed, attempt=" + attempt + ": " + ex.getMessage())
	    } finally {
	        if (connection != null) {
	            connection.disconnect()
	        }
	    }
	    if (!uploaded && attempt < 3) {
	        sleep(2000L * attempt)
	    }
	}`

	return `<PostThreadGroup guiclass="PostThreadGroupGui" testclass="PostThreadGroup" testname="错误明细上传" enabled="true">
  <stringProp name="ThreadGroup.on_sample_error">continue</stringProp>
  <elementProp name="ThreadGroup.main_controller" elementType="LoopController" guiclass="LoopControlPanel" testclass="LoopController" testname="循环控制器" enabled="true">
    <boolProp name="LoopController.continue_forever">false</boolProp>
    <intProp name="LoopController.loops">1</intProp>
  </elementProp>
  <stringProp name="ThreadGroup.num_threads">1</stringProp>
  <stringProp name="ThreadGroup.ramp_time">1</stringProp>
  <longProp name="ThreadGroup.start_time">0</longProp>
  <longProp name="ThreadGroup.end_time">0</longProp>
  <boolProp name="ThreadGroup.scheduler">false</boolProp>
  <boolProp name="ThreadGroup.same_user_on_next_iteration">true</boolProp>
  <stringProp name="ThreadGroup.delay"></stringProp>
  <stringProp name="ThreadGroup.duration"></stringProp>
</PostThreadGroup>
<hashTree>
  <JSR223Sampler guiclass="TestBeanGUI" testclass="JSR223Sampler" testname="上传错误明细文件" enabled="true">
    <stringProp name="cacheKey">true</stringProp>
    <stringProp name="filename"></stringProp>
    <stringProp name="parameters"></stringProp>
    <stringProp name="script">` + xmlEscape(script) + `</stringProp>
    <stringProp name="scriptLanguage">groovy</stringProp>
  </JSR223Sampler>
  <hashTree/>
</hashTree>`
}

func createRuntimeJMXWithErrorDetailListener(sourcePath, targetPath string) error {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	xmlContent := absolutizeRuntimeDependencyPaths(string(content), filepath.Dir(sourcePath))
	insertPos, err := findTestPlanHashTreeInsertPos(xmlContent)
	if err != nil {
		return err
	}
	injected := xmlContent[:insertPos] + buildErrorDetailListenerXML() + "\n" + buildErrorDetailUploadThreadGroupXML() + "\n" + xmlContent[insertPos:]
	return os.WriteFile(targetPath, []byte(injected), 0644)
}

func findTestPlanHashTreeInsertPos(xmlContent string) (int, error) {
	testPlanEnd := strings.Index(xmlContent, "</TestPlan>")
	if testPlanEnd == -1 {
		return 0, fmt.Errorf("未找到 TestPlan 节点")
	}
	searchStart := testPlanEnd + len("</TestPlan>")
	hashTreeOpenRel := strings.Index(xmlContent[searchStart:], "<hashTree")
	if hashTreeOpenRel == -1 {
		return 0, fmt.Errorf("未找到 TestPlan 对应的 hashTree")
	}
	hashTreeOpen := searchStart + hashTreeOpenRel
	openTagEndRel := strings.Index(xmlContent[hashTreeOpen:], ">")
	if openTagEndRel == -1 {
		return 0, fmt.Errorf("hashTree 开始标签不完整")
	}
	pos := hashTreeOpen + openTagEndRel + 1
	depth := 1
	for pos < len(xmlContent) {
		nextOpenRel := strings.Index(xmlContent[pos:], "<hashTree")
		nextCloseRel := strings.Index(xmlContent[pos:], "</hashTree>")
		if nextCloseRel == -1 {
			return 0, fmt.Errorf("未找到 TestPlan hashTree 结束位置")
		}
		nextClose := pos + nextCloseRel
		nextOpen := -1
		if nextOpenRel != -1 {
			nextOpen = pos + nextOpenRel
		}
		if nextOpen != -1 && nextOpen < nextClose {
			openEndRel := strings.Index(xmlContent[nextOpen:], ">")
			if openEndRel == -1 {
				return 0, fmt.Errorf("hashTree 子节点开始标签不完整")
			}
			openEnd := nextOpen + openEndRel + 1
			tagText := xmlContent[nextOpen:openEnd]
			if !strings.HasSuffix(strings.TrimSpace(tagText), "/>") {
				depth++
			}
			pos = openEnd
			continue
		}
		depth--
		if depth == 0 {
			return nextClose, nil
		}
		pos = nextClose + len("</hashTree>")
	}
	return 0, fmt.Errorf("未能定位 TestPlan hashTree 插入点")
}

func buildErrorDetailKey(timestamp, label, threadName string, elapsed int64, url string) string {
	return strings.Join([]string{
		strings.TrimSpace(timestamp),
		strings.TrimSpace(label),
		strings.TrimSpace(threadName),
		strconv.FormatInt(elapsed, 10),
		strings.TrimSpace(url),
	}, "|")
}

func loadErrorDetailEntries(path string) (map[string][]errorDetailEntry, int) {
	file, err := os.Open(path)
	if err != nil {
		return map[string][]errorDetailEntry{}, 0
	}
	defer file.Close()

	entries := make(map[string][]errorDetailEntry)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	total := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry errorDetailEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		key := buildErrorDetailKey(entry.Timestamp, entry.Label, entry.ThreadName, entry.Elapsed, entry.URL)
		entries[key] = append(entries[key], entry)
		total++
	}
	return entries, total
}

func loadAllErrorDetailEntries(resultPath string) (map[string][]errorDetailEntry, int) {
	allEntries := make(map[string][]errorDetailEntry)
	total := 0
	merge := func(entries map[string][]errorDetailEntry, count int) {
		for key, list := range entries {
			allEntries[key] = append(allEntries[key], list...)
		}
		total += count
	}

	baseDir := filepath.Dir(resultPath)
	localEntries, localCount := loadErrorDetailEntries(filepath.Join(baseDir, "error-details.ndjson"))
	merge(localEntries, localCount)

	detailDir := filepath.Join(baseDir, "error-details")
	files, err := filepath.Glob(filepath.Join(detailDir, "*.ndjson"))
	if err == nil {
		sort.Strings(files)
		for _, file := range files {
			entries, count := loadErrorDetailEntries(file)
			merge(entries, count)
		}
	}

	return allEntries, total
}

func SaveUploadedExecutionErrorDetails(execID int64, token, source, content string) error {
	execution, err := GetExecution(execID)
	if err != nil {
		return fmt.Errorf("执行记录不存在")
	}
	if execution.ResultPath == "" {
		return fmt.Errorf("执行结果目录未初始化")
	}

	resultDir := filepath.Dir(execution.ResultPath)
	expectedToken, err := readExecutionUploadToken(resultDir)
	if err != nil {
		return fmt.Errorf("当前执行未开启错误明细上传")
	}
	if strings.TrimSpace(token) == "" || token != expectedToken {
		return fmt.Errorf("错误明细上传令牌无效")
	}

	source = sanitizeDetailSourceName(source)
	detailDir := filepath.Join(resultDir, "error-details")
	if err := os.MkdirAll(detailDir, 0755); err != nil {
		return fmt.Errorf("创建错误明细目录失败: %w", err)
	}

	targetFile := filepath.Join(detailDir, source+".ndjson")
	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("保存错误明细文件失败: %w", err)
	}
	go refreshExecutionErrorAnalysisIndex(execID, nil)
	return nil
}

func generateExecutionUploadToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func saveExecutionUploadToken(resultDir, token string) error {
	return os.WriteFile(filepath.Join(resultDir, "error-details.token"), []byte(token), 0600)
}

func readExecutionUploadToken(resultDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(resultDir, "error-details.token"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func sanitizeDetailSourceName(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_", "..", "_")
	source = replacer.Replace(source)
	if len(source) > 80 {
		source = source[:80]
	}
	return source
}

func getExecutionExpectedDetailSources(execution *model.Execution) []string {
	if execution == nil || strings.TrimSpace(execution.SlaveIDs) == "" {
		return nil
	}
	var slaveIDs []int64
	if err := json.Unmarshal([]byte(execution.SlaveIDs), &slaveIDs); err != nil || len(slaveIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(slaveIDs))
	args := make([]interface{}, 0, len(slaveIDs))
	for _, id := range slaveIDs {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	query := fmt.Sprintf("SELECT name, host FROM slaves WHERE id IN (%s) ORDER BY id ASC", strings.Join(placeholders, ","))
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sources []string
	for rows.Next() {
		var name, host string
		if err := rows.Scan(&name, &host); err != nil {
			continue
		}
		display := strings.TrimSpace(host)
		if strings.TrimSpace(name) != "" {
			display = fmt.Sprintf("%s (%s)", host, name)
		}
		sources = append(sources, display)
	}
	return sources
}

func sourceMatchesExpected(expected, received string) bool {
	expected = strings.TrimSpace(expected)
	received = strings.TrimSpace(received)
	if expected == "" || received == "" {
		return false
	}
	extractHost := func(value string) string {
		value = strings.TrimSpace(value)
		if value == "" {
			return ""
		}
		if idx := strings.Index(value, "("); idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
		if idx := strings.IndexAny(value, " \t"); idx > 0 {
			value = strings.TrimSpace(value[:idx])
		}
		return value
	}
	expectedHost := extractHost(expected)
	receivedHost := extractHost(received)
	if expectedHost != "" && receivedHost != "" && expectedHost == receivedHost {
		return true
	}
	if strings.Contains(received, expected) {
		return true
	}
	if idx := strings.Index(expected, " "); idx > 0 {
		host := strings.TrimSpace(expected[:idx])
		return host != "" && strings.Contains(received, host)
	}
	return false
}

// truncateToTenSeconds 将时间戳字符串截取到10秒级别（格式: "15:04:05"）
// 支持 epoch 毫秒数和日期时间字符串两种格式
func truncateToTenSeconds(timestampStr string) string {
	if timestampStr == "" {
		return "unknown"
	}
	// 尝试解析为 epoch 毫秒
	if ms, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		t := time.Unix(ms/1000, (ms%1000)*1e6)
		// 截断到10秒
		t = t.Truncate(10 * time.Second)
		return t.Format("15:04:05")
	}
	// 尝试解析为日期时间字符串（带毫秒）
	if t, err := time.Parse("2006-01-02 15:04:05.000", timestampStr); err == nil {
		t = t.Truncate(10 * time.Second)
		return t.Format("15:04:05")
	}
	// 尝试解析为日期时间字符串（不带毫秒）
	if t, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
		t = t.Truncate(10 * time.Second)
		return t.Format("15:04:05")
	}
	return "unknown"
}

// truncateToMinute 将时间戳字符串截取到分钟级别（格式: "15:04"）
// 支持 epoch 毫秒数和日期时间字符串两种格式
// 已弃用：请使用 truncateToTenSeconds
func truncateToMinute(timestampStr string) string {
	if timestampStr == "" {
		return "unknown"
	}
	// 尝试解析为 epoch 毫秒
	if ms, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
		t := time.Unix(ms/1000, 0)
		return t.Format("15:04")
	}
	// 尝试解析为日期时间字符串（带毫秒）
	if t, err := time.Parse("2006-01-02 15:04:05.000", timestampStr); err == nil {
		return t.Format("15:04")
	}
	// 尝试解析为日期时间字符串（不带毫秒）
	if t, err := time.Parse("2006-01-02 15:04:05", timestampStr); err == nil {
		return t.Format("15:04")
	}
	return "unknown"
}

func buildEmptyErrorAnalysis(execution *model.Execution, hint string) *ErrorAnalysis {
	expectedDetailSources := getExecutionExpectedDetailSources(execution)
	return &ErrorAnalysis{
		TotalErrors:           0,
		TotalSamples:          0,
		ErrorRate:             0,
		ErrorTypes:            []ErrorType{},
		Records:               []ErrorRecord{},
		Truncated:             false,
		TypeTruncated:         map[string]bool{},
		DetailFieldsAvailable: false,
		DetailStorageHint:     hint,
		AvailableDetailFields: []string{},
		ExpectedDetailSources: expectedDetailSources,
		ReceivedDetailSources: []string{},
		MissingDetailSources:  []string{},
		DetailUploadWarning:   "",
		CategoryDistribution:  []ErrorCluster{},
		SourceDistribution:    []ErrorCluster{},
		APIDistribution:       []ErrorCluster{},
		ReportHighlights:      []string{},
	}
}

func normalizeErrorSourceLabel(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "未知来源"
	}
	return source
}

func classifyErrorCategory(responseCode, responseMessage, failureMessage string) (string, string, string) {
	normalizedCode := strings.TrimSpace(strings.ToLower(responseCode))
	normalizedMessage := strings.TrimSpace(strings.ToLower(responseMessage))
	normalizedFailure := strings.TrimSpace(strings.ToLower(failureMessage))
	combined := strings.Join([]string{normalizedCode, normalizedMessage, normalizedFailure}, " ")

	switch {
	case strings.Contains(combined, "csvdataset") ||
		strings.Contains(combined, "fileserver") ||
		strings.Contains(combined, "must exist and be readable") ||
		strings.Contains(combined, "could not read file header line"):
		return "csv_dependency", "CSV/文件依赖", "warning"
	case strings.Contains(combined, "jsr223") ||
		strings.Contains(combined, "groovy") ||
		strings.Contains(combined, "beanshell") ||
		strings.Contains(combined, "scriptexception"):
		return "script_runtime", "脚本运行异常", "danger"
	case strings.Contains(combined, "assert") ||
		strings.Contains(combined, "expected to contain") ||
		strings.Contains(combined, "test failed:"):
		if normalizedCode == "200" || normalizedCode == "" {
			return "business_assertion", "业务断言失败", "danger"
		}
		return "assertion", "断言失败", "danger"
	case strings.Contains(combined, "read timed out") ||
		strings.Contains(combined, "sockettimeoutexception") ||
		strings.Contains(combined, "response timeout"):
		return "response_timeout", "响应超时", "warning"
	case strings.Contains(combined, "connect timed out") ||
		strings.Contains(combined, "connection refused") ||
		strings.Contains(combined, "unknownhost") ||
		strings.Contains(combined, "no route to host") ||
		strings.Contains(combined, "broken pipe") ||
		strings.Contains(combined, "connection reset") ||
		strings.Contains(combined, "non http response code"):
		return "network", "网络连接异常", "warning"
	case strings.HasPrefix(normalizedCode, "5"):
		return "http_5xx", "服务端错误", "danger"
	case strings.HasPrefix(normalizedCode, "4"):
		return "http_4xx", "客户端错误", "warning"
	case normalizedCode == "200" && normalizedFailure != "":
		return "business_failure", "业务失败", "danger"
	default:
		return "other", "其他失败", "warning"
	}
}

func summarizeClusterTopKeys(counts map[string]int, limit int) []string {
	if len(counts) == 0 || limit <= 0 {
		return nil
	}
	type item struct {
		Key   string
		Count int
	}
	items := make([]item, 0, len(counts))
	for key, count := range counts {
		items = append(items, item{Key: key, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	result := make([]string, 0, len(items))
	for _, current := range items {
		result = append(result, current.Key)
	}
	return result
}

func buildErrorAnalysisOverview(analysis *ErrorAnalysis) *ErrorAnalysisOverview {
	if analysis == nil {
		return nil
	}
	return &ErrorAnalysisOverview{
		TotalErrors:              analysis.TotalErrors,
		TotalSamples:             analysis.TotalSamples,
		ErrorRate:                analysis.ErrorRate,
		ErrorTypes:               analysis.ErrorTypes,
		Truncated:                analysis.Truncated,
		TypeTruncated:            analysis.TypeTruncated,
		DetailFieldsAvailable:    analysis.DetailFieldsAvailable,
		DetailStorageHint:        analysis.DetailStorageHint,
		AvailableDetailFields:    analysis.AvailableDetailFields,
		ExpectedDetailSources:    analysis.ExpectedDetailSources,
		ReceivedDetailSources:    analysis.ReceivedDetailSources,
		MissingDetailSources:     analysis.MissingDetailSources,
		DetailUploadWarning:      analysis.DetailUploadWarning,
		ResponseCodeDistribution: analysis.ResponseCodeDistribution,
		ErrorTimeline:            analysis.ErrorTimeline,
		TopErrorMessages:         analysis.TopErrorMessages,
		CategoryDistribution:     analysis.CategoryDistribution,
		SourceDistribution:       analysis.SourceDistribution,
		APIDistribution:          analysis.APIDistribution,
		ReportHighlights:         analysis.ReportHighlights,
	}
}

func buildErrorReportHighlights(analysis *ErrorAnalysis) []string {
	if analysis == nil || analysis.TotalErrors == 0 {
		return []string{"当前执行没有错误样本，可直接进入指标复盘阶段。"}
	}

	highlights := make([]string, 0, 5)
	if len(analysis.CategoryDistribution) > 0 {
		top := analysis.CategoryDistribution[0]
		highlights = append(highlights, fmt.Sprintf("错误主要集中在“%s”，共 %d 条，占全部错误 %.2f%%。", top.Label, top.Count, top.Percentage))
	}
	if len(analysis.APIDistribution) > 0 {
		top := analysis.APIDistribution[0]
		target := top.Label
		if strings.TrimSpace(target) == "" {
			target = top.URL
		}
		highlights = append(highlights, fmt.Sprintf("最需要优先排查的接口是“%s”，累计失败 %d 次。", target, top.Count))
	}
	if len(analysis.SourceDistribution) > 0 {
		top := analysis.SourceDistribution[0]
		if top.Label != "未知来源" {
			highlights = append(highlights, fmt.Sprintf("错误来源最集中的节点是 %s，占错误总量 %.2f%%。", top.Label, top.Percentage))
		}
	}
	if analysis.DetailUploadWarning != "" {
		highlights = append(highlights, analysis.DetailUploadWarning)
	}
	if len(analysis.TopErrorMessages) > 0 {
		top := analysis.TopErrorMessages[0]
		highlights = append(highlights, fmt.Sprintf("Top 失败信息为“%s”，出现 %d 次。", top.Message, top.Count))
	}
	if len(highlights) == 0 {
		highlights = append(highlights, fmt.Sprintf("当前共识别 %d 条错误记录，建议结合错误类型分布继续排查。", analysis.TotalErrors))
	}
	return highlights
}

func BuildExecutionErrorReportMarkdown(execution *model.Execution, analysis *ErrorAnalysis) string {
	title := "执行错误报告"
	if execution != nil && strings.TrimSpace(execution.ScriptName) != "" {
		title = fmt.Sprintf("执行错误报告 - %s", execution.ScriptName)
	}

	lines := []string{
		"# " + title,
		"",
		fmt.Sprintf("- 生成时间：%s", time.Now().Format("2006-01-02 15:04:05")),
	}
	if execution != nil {
		lines = append(lines, fmt.Sprintf("- 执行 ID：%d", execution.ID))
		lines = append(lines, fmt.Sprintf("- 执行状态：%s", execution.Status))
		lines = append(lines, fmt.Sprintf("- 开始时间：%s", strings.TrimSpace(execution.StartTime)))
		lines = append(lines, fmt.Sprintf("- 结束时间：%s", strings.TrimSpace(execution.EndTime)))
	}
	lines = append(lines, "")

	if analysis == nil {
		lines = append(lines, "当前没有可用的错误分析数据。")
		return strings.Join(lines, "\n")
	}

	lines = append(lines,
		"## 概览",
		"",
		fmt.Sprintf("- 总样本数：%d", analysis.TotalSamples),
		fmt.Sprintf("- 错误总数：%d", analysis.TotalErrors),
		fmt.Sprintf("- 错误率：%.2f%%", analysis.ErrorRate),
		"",
		"## 复盘结论",
		"",
	)
	for _, item := range analysis.ReportHighlights {
		lines = append(lines, "- "+item)
	}

	appendClusterSection := func(title string, items []ErrorCluster) {
		if len(items) == 0 {
			return
		}
		lines = append(lines, "", "## "+title, "")
		limit := len(items)
		if limit > 10 {
			limit = 10
		}
		for idx := 0; idx < limit; idx++ {
			current := items[idx]
			line := fmt.Sprintf("- %s：%d 条，占比 %.2f%%", current.Label, current.Count, current.Percentage)
			if current.URL != "" {
				line += fmt.Sprintf("，URL: %s", current.URL)
			}
			if current.Example != "" {
				line += fmt.Sprintf("，样例: %s", current.Example)
			}
			lines = append(lines, line)
		}
	}

	appendClusterSection("错误分类聚类", analysis.CategoryDistribution)
	appendClusterSection("错误来源节点", analysis.SourceDistribution)
	appendClusterSection("失败接口 Top", analysis.APIDistribution)

	if len(analysis.TopErrorMessages) > 0 {
		lines = append(lines, "", "## Top 错误信息", "")
		for _, item := range analysis.TopErrorMessages {
			lines = append(lines, fmt.Sprintf("- %s：%d 次", item.Message, item.Count))
		}
	}

	if analysis.DetailUploadWarning != "" {
		lines = append(lines, "", "## 明细链路提示", "", "- "+analysis.DetailUploadWarning)
		if len(analysis.MissingDetailSources) > 0 {
			lines = append(lines, fmt.Sprintf("- 未回传节点：%s", strings.Join(analysis.MissingDetailSources, "、")))
		}
	}

	return strings.Join(lines, "\n")
}

func statSignaturePart(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return path + "|missing"
	}
	return fmt.Sprintf("%s|%d|%d", path, info.Size(), info.ModTime().UnixNano())
}

func buildErrorAnalysisSignature(resultPath string) string {
	parts := []string{statSignaturePart(resultPath)}
	baseDir := filepath.Dir(resultPath)
	parts = append(parts, statSignaturePart(filepath.Join(baseDir, "error-details.ndjson")))
	files, err := filepath.Glob(filepath.Join(baseDir, "error-details", "*.ndjson"))
	if err == nil {
		sort.Strings(files)
		for _, file := range files {
			parts = append(parts, statSignaturePart(file))
		}
	}
	return strings.Join(parts, "||")
}

func getCachedErrorAnalysis(execID int64, signature string) *ErrorAnalysis {
	errorAnalysisCacheMu.RLock()
	entry, ok := errorAnalysisCache[execID]
	errorAnalysisCacheMu.RUnlock()
	if !ok {
		return nil
	}
	if entry.Signature != signature || time.Now().After(entry.ExpiresAt) {
		return nil
	}
	return entry.Analysis
}

func setCachedErrorAnalysis(execID int64, signature string, analysis *ErrorAnalysis) {
	errorAnalysisCacheMu.Lock()
	errorAnalysisCache[execID] = errorAnalysisCacheEntry{
		Signature: signature,
		ExpiresAt: time.Now().Add(errorAnalysisCacheTTL),
		Analysis:  analysis,
	}
	errorAnalysisCacheMu.Unlock()
}

// GetExecutionErrors 获取执行错误记录
func GetExecutionErrors(execID int64) (*ErrorAnalysis, error) {
	// 1. 查询执行记录获取 result_path
	execution, _ := GetExecution(execID)
	var resultPath string
	err := database.DB.QueryRow("SELECT result_path FROM executions WHERE id = ?", execID).Scan(&resultPath)
	if err != nil {
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}
	if resultPath == "" {
		return buildEmptyErrorAnalysis(execution, "当前执行尚未生成结果文件，请稍后刷新。"), nil
	}

	resultPaths := discoverExecutionResultPaths(resultPath)
	if len(resultPaths) > 1 {
		if info, statErr := os.Stat(resultPath); statErr != nil || info.Size() == 0 {
			if mergeErr := mergeJTLFiles(resultPaths, resultPath); mergeErr != nil {
				fmt.Printf("[JTL合并][警告] 错误分析自动合并失败: %v\n", mergeErr)
				resultPath = resultPaths[0]
			}
		}
	}
	if len(resultPaths) > 0 {
		if info, statErr := os.Stat(resultPath); statErr != nil || info.Size() == 0 {
			resultPath = resultPaths[0]
		}
	}

	cacheSignature := buildErrorAnalysisSignature(resultPath)
	if cached := getCachedErrorAnalysis(execID, cacheSignature); cached != nil {
		return cached, nil
	}
	if indexed, ok := loadIndexedErrorAnalysis(resultPath, cacheSignature); ok {
		setCachedErrorAnalysis(execID, cacheSignature, indexed)
		return indexed, nil
	}

	// 2. 打开并解析 JTL 文件
	file, err := os.Open(resultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return buildEmptyErrorAnalysis(execution, "当前执行结果文件尚未落盘，请稍后刷新。"), nil
		}
		return nil, fmt.Errorf("打开结果文件失败: %w", err)
	}
	defer file.Close()

	if info, statErr := file.Stat(); statErr == nil && info.Size() == 0 {
		return buildEmptyErrorAnalysis(execution, "当前执行结果文件还在写入中，请稍后刷新。"), nil
	}
	detailEntries, detailEntryCount := loadAllErrorDetailEntries(resultPath)

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // 允许字段数不一致

	// 3. 读取表头
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return buildEmptyErrorAnalysis(execution, "当前执行结果文件还在初始化，请稍后刷新。"), nil
		}
		return nil, fmt.Errorf("读取表头失败: %w", err)
	}

	// 打印表头信息便于调试
	fmt.Printf("[JTL] 表头列数: %d, 列名: %v\n", len(headers), headers)

	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[strings.TrimSpace(h)] = i
	}

	headerCount := len(headers)
	detailCandidates := []string{
		"requestHeaders",
		"request_headers",
		"samplerData",
		"sampler_data",
		"queryString",
		"responseHeaders",
		"response_headers",
		"responseData.onError",
		"responseData",
		"response_data",
	}
	availableDetailFields := make([]string, 0, len(detailCandidates))
	for _, field := range detailCandidates {
		if _, ok := colIndex[field]; ok {
			availableDetailFields = append(availableDetailFields, field)
		}
	}
	receivedDetailSourcesSet := make(map[string]struct{})
	for _, list := range detailEntries {
		for _, entry := range list {
			source := strings.TrimSpace(entry.Source)
			if source != "" {
				receivedDetailSourcesSet[source] = struct{}{}
			}
		}
	}
	receivedDetailSources := make([]string, 0, len(receivedDetailSourcesSet))
	for source := range receivedDetailSourcesSet {
		receivedDetailSources = append(receivedDetailSources, source)
	}
	sort.Strings(receivedDetailSources)

	expectedDetailSources := getExecutionExpectedDetailSources(execution)
	missingDetailSources := make([]string, 0)
	for _, expected := range expectedDetailSources {
		matched := false
		for _, received := range receivedDetailSources {
			if sourceMatchesExpected(expected, received) {
				matched = true
				break
			}
		}
		if !matched {
			missingDetailSources = append(missingDetailSources, expected)
		}
	}

	detailFieldsAvailable := len(availableDetailFields) > 0 || detailEntryCount > 0
	detailStorageHint := ""
	detailUploadWarning := ""
	if detailEntryCount > 0 {
		availableDetailFields = append(availableDetailFields,
			"listener.request_headers",
			"listener.request_body",
			"listener.response_headers",
			"listener.response_data",
		)
		detailStorageHint = "当前执行已通过错误明细监听器保存失败样本的 HTTP 请求/响应信息，详情页会优先读取这份明细结果。"
	} else if len(availableDetailFields) > 0 {
		detailStorageHint = "当前结果文件已包含请求/响应详情字段，若个别错误样本为空，通常是该样本本身未写出对应内容。"
	} else {
		detailStorageHint = "当前结果文件未保存请求/响应详情字段，仅能展示 URL、状态码、失败原因、耗时和字节数。后续执行请使用新版执行配置重新跑一次。"
	}
	if len(missingDetailSources) > 0 {
		detailUploadWarning = "部分节点未回传错误明细，可能存在上传失败或明细数据丢失。"
	}

	// 4. 遍历所有记录，收集错误
	const maxRecordsPerType = 10000
	const maxSamplesPerType = 5

	type clusterAccumulator struct {
		Label        string
		Tone         string
		Example      string
		URL          string
		Count        int
		LabelCounts  map[string]int
		SourceCounts map[string]int
	}

	var totalSamples int
	var totalErrors int

	// 错误类型映射: key = "label|responseCode"
	errorTypeMap := make(map[string]*ErrorType)
	var errorTypeOrder []string // 保持出现顺序

	// 错误记录
	var records []ErrorRecord
	// 每种错误类型已采集的记录数
	typeCountMap := make(map[string]int)
	// 记录哪些类型被截断
	typeTruncated := make(map[string]bool)

	// 新增收集变量
	codeCountMap := make(map[string]int)           // responseCode → count
	timelineMap := make(map[string]*TimelinePoint) // "HH:MM:SS" → point (10秒粒度)
	messageCounts := make(map[string]int)          // failureMessage → count
	totalSamplesByBucket := make(map[string]int)   // 每个时间桶总样本数
	categoryMap := make(map[string]*clusterAccumulator)
	sourceMap := make(map[string]*clusterAccumulator)
	apiMap := make(map[string]*clusterAccumulator)

	getField := func(record []string, field string) string {
		if idx, ok := colIndex[strings.TrimSpace(field)]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	getFirstField := func(record []string, fields ...string) string {
		for _, field := range fields {
			if value := getField(record, field); value != "" {
				return value
			}
		}
		return ""
	}

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		// 跳过列数异常的行（可能是多行字段的残行）
		if len(record) < headerCount/2 {
			continue
		}

		label := getField(record, "label")
		responseCode := getField(record, "responseCode")
		responseMessage := getField(record, "responseMessage")
		url := getField(record, "URL")
		requestSample := isRequestSample(url)
		transactionSample := isTransactionSample(label, url, responseMessage)
		baseSample := requestSample
		if !requestSample && transactionSample {
			baseSample = true
		}
		if !baseSample {
			continue
		}

		totalSamples++

		// 收集时间戳用于时间线统计
		tsStr := getField(record, "timeStamp")
		timeBucket := truncateToTenSeconds(tsStr)
		if timeBucket != "unknown" {
			totalSamplesByBucket[timeBucket]++
		}

		success := getField(record, "success")
		if strings.ToLower(success) != "false" {
			continue
		}
		totalErrors++

		if totalErrors == 1 && !detailFieldsAvailable {
			fmt.Printf("[JTL] 警告: JTL 文件不包含请求/响应详情字段，错误详情将部分缺失。可用列: %v\n", headers)
		}

		// 收集错误统计数据
		codeCountMap[responseCode]++
		if timeBucket != "unknown" {
			if point, ok := timelineMap[timeBucket]; ok {
				point.ErrorCount++
			} else {
				timelineMap[timeBucket] = &TimelinePoint{
					TimeBucket: timeBucket,
					ErrorCount: 1,
				}
			}
		}
		failureMessage := getField(record, "failureMessage")
		if failureMessage != "" {
			messageCounts[failureMessage]++
		}
		responseData := getFirstField(record, "responseData.onError", "responseData", "response_data")
		responseHeaders := getFirstField(record, "responseHeaders", "response_headers")
		threadName := getField(record, "threadName")

		elapsed, _ := strconv.ParseInt(getField(record, "elapsed"), 10, 64)
		bytes, _ := strconv.ParseInt(getField(record, "bytes"), 10, 64)

		// 提取新增字段
		requestHeaders := getFirstField(record, "requestHeaders", "request_headers")
		requestBody := getFirstField(record, "samplerData", "sampler_data", "queryString")
		sentBytesStr := getFirstField(record, "sentBytes", "sent_bytes")
		latencyStr := getFirstField(record, "Latency", "latency")
		connectStr := getFirstField(record, "Connect", "connect")

		sentBytes, _ := strconv.ParseInt(sentBytesStr, 10, 64)
		latency, _ := strconv.ParseInt(latencyStr, 10, 64)
		connectTime, _ := strconv.ParseInt(connectStr, 10, 64)

		detailKey := buildErrorDetailKey(tsStr, label, threadName, elapsed, url)
		detailSource := ""
		if matches := detailEntries[detailKey]; len(matches) > 0 {
			detail := matches[0]
			detailEntries[detailKey] = matches[1:]
			detailSource = detail.Source
			if detail.URL != "" {
				url = detail.URL
			}
			if detail.ResponseMessage != "" {
				responseMessage = detail.ResponseMessage
			}
			if detail.FailureMessage != "" {
				failureMessage = detail.FailureMessage
			}
			if detail.RequestHeaders != "" {
				requestHeaders = detail.RequestHeaders
			}
			if detail.RequestBody != "" {
				requestBody = detail.RequestBody
			}
			if detail.ResponseHeaders != "" {
				responseHeaders = detail.ResponseHeaders
			}
			if detail.ResponseData != "" {
				responseData = detail.ResponseData
			}
		}

		timestamp := ""
		if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
			timestamp = time.Unix(ts/1000, (ts%1000)*1e6).Format("2006-01-02 15:04:05")
		}

		errRecord := ErrorRecord{
			Source:          detailSource,
			Timestamp:       timestamp,
			Elapsed:         elapsed,
			Label:           label,
			ResponseCode:    responseCode,
			ResponseMessage: responseMessage,
			ResponseData:    responseData,
			ResponseHeaders: responseHeaders,
			ThreadName:      threadName,
			FailureMessage:  failureMessage,
			URL:             url,
			Bytes:           bytes,
			RequestHeaders:  requestHeaders,
			RequestBody:     requestBody,
			SentBytes:       sentBytes,
			Latency:         latency,
			ConnectTime:     connectTime,
		}

		categoryKey, categoryLabel, categoryTone := classifyErrorCategory(responseCode, responseMessage, failureMessage)
		sourceLabel := normalizeErrorSourceLabel(detailSource)
		apiKey := strings.TrimSpace(label)
		if strings.TrimSpace(url) != "" {
			apiKey += "|" + strings.TrimSpace(url)
		}
		if strings.TrimSpace(apiKey) == "" {
			apiKey = strings.TrimSpace(url)
		}

		ensureCluster := func(store map[string]*clusterAccumulator, key, label, tone string) *clusterAccumulator {
			if current, ok := store[key]; ok {
				return current
			}
			current := &clusterAccumulator{
				Label:        label,
				Tone:         tone,
				LabelCounts:  make(map[string]int),
				SourceCounts: make(map[string]int),
			}
			store[key] = current
			return current
		}

		categoryCluster := ensureCluster(categoryMap, categoryKey, categoryLabel, categoryTone)
		categoryCluster.Count++
		categoryCluster.LabelCounts[label]++
		categoryCluster.SourceCounts[sourceLabel]++
		if categoryCluster.Example == "" {
			categoryCluster.Example = failureMessage
		}

		sourceCluster := ensureCluster(sourceMap, sourceLabel, sourceLabel, "warning")
		sourceCluster.Count++
		sourceCluster.LabelCounts[label]++
		if sourceCluster.Example == "" {
			sourceCluster.Example = failureMessage
		}

		apiCluster := ensureCluster(apiMap, apiKey, label, "danger")
		apiCluster.Count++
		apiCluster.URL = url
		apiCluster.SourceCounts[sourceLabel]++
		if apiCluster.Example == "" {
			apiCluster.Example = failureMessage
		}

		// 更新错误类型统计
		typeKey := label + "|" + responseCode
		if et, ok := errorTypeMap[typeKey]; ok {
			et.Count++
			et.LastTime = timestamp
			if len(et.SampleErrors) < maxSamplesPerType {
				et.SampleErrors = append(et.SampleErrors, errRecord)
			}
		} else {
			et := &ErrorType{
				Label:           label,
				ResponseCode:    responseCode,
				ResponseMessage: responseMessage,
				Category:        categoryLabel,
				ResponsePreview: buildResponsePreview(responseData, failureMessage),
				Count:           1,
				FirstTime:       timestamp,
				LastTime:        timestamp,
				SampleErrors:    []ErrorRecord{errRecord},
			}
			errorTypeMap[typeKey] = et
			errorTypeOrder = append(errorTypeOrder, typeKey)
		}

		// 按错误类型独立计数截断
		// 每种类型最多记录 maxRecordsPerType 条
		if typeCountMap[typeKey] >= maxRecordsPerType {
			typeTruncated[typeKey] = true
			continue
		}
		records = append(records, errRecord)
		typeCountMap[typeKey]++
	}

	// 5. 构建结果
	errorTypes := make([]ErrorType, 0, len(errorTypeOrder))
	for _, key := range errorTypeOrder {
		et := errorTypeMap[key]
		if totalErrors > 0 {
			et.Percentage = float64(et.Count) * 100.0 / float64(totalErrors)
		}
		errorTypes = append(errorTypes, *et)
	}

	// 按数量降序排序
	sort.Slice(errorTypes, func(i, j int) bool {
		return errorTypes[i].Count > errorTypes[j].Count
	})

	buildClusters := func(store map[string]*clusterAccumulator) []ErrorCluster {
		clusters := make([]ErrorCluster, 0, len(store))
		for key, current := range store {
			percentage := 0.0
			if totalErrors > 0 {
				percentage = float64(current.Count) * 100.0 / float64(totalErrors)
			}
			clusters = append(clusters, ErrorCluster{
				Key:        key,
				Label:      current.Label,
				Count:      current.Count,
				Percentage: percentage,
				Tone:       current.Tone,
				Example:    current.Example,
				URL:        current.URL,
				TopLabels:  summarizeClusterTopKeys(current.LabelCounts, 3),
				TopSources: summarizeClusterTopKeys(current.SourceCounts, 3),
			})
		}
		sort.Slice(clusters, func(i, j int) bool {
			if clusters[i].Count == clusters[j].Count {
				return clusters[i].Label < clusters[j].Label
			}
			return clusters[i].Count > clusters[j].Count
		})
		return clusters
	}

	errorRate := 0.0
	if totalSamples > 0 {
		errorRate = float64(totalErrors) * 100.0 / float64(totalSamples)
	}

	// 构建响应码分布统计
	responseCodeDistribution := make([]CodeCount, 0, len(codeCountMap))
	for code, count := range codeCountMap {
		percentage := 0.0
		if totalErrors > 0 {
			percentage = float64(count) * 100.0 / float64(totalErrors)
		}
		responseCodeDistribution = append(responseCodeDistribution, CodeCount{
			Code:       code,
			Count:      count,
			Percentage: percentage,
		})
	}
	// 按 count 降序排序
	sort.Slice(responseCodeDistribution, func(i, j int) bool {
		return responseCodeDistribution[i].Count > responseCodeDistribution[j].Count
	})

	// 构建错误时间线统计
	errorTimeline := make([]TimelinePoint, 0, len(timelineMap))
	for bucket, point := range timelineMap {
		point.SampleCount = totalSamplesByBucket[bucket]
		if point.SampleCount > 0 {
			point.ErrorRate = float64(point.ErrorCount) * 100.0 / float64(point.SampleCount)
		}
		errorTimeline = append(errorTimeline, *point)
	}
	// 按时间排序
	sort.Slice(errorTimeline, func(i, j int) bool {
		return errorTimeline[i].TimeBucket < errorTimeline[j].TimeBucket
	})

	// 构建 Top 错误信息统计
	topErrorMessages := make([]MessageCount, 0, len(messageCounts))
	for msg, count := range messageCounts {
		topErrorMessages = append(topErrorMessages, MessageCount{
			Message: msg,
			Count:   count,
		})
	}
	// 按 count 降序排序
	sort.Slice(topErrorMessages, func(i, j int) bool {
		return topErrorMessages[i].Count > topErrorMessages[j].Count
	})
	// 只保留前 10 条
	if len(topErrorMessages) > 10 {
		topErrorMessages = topErrorMessages[:10]
	}

	categoryDistribution := buildClusters(categoryMap)
	sourceDistribution := buildClusters(sourceMap)
	apiDistribution := buildClusters(apiMap)

	analysis := &ErrorAnalysis{
		TotalErrors:              totalErrors,
		TotalSamples:             totalSamples,
		ErrorRate:                errorRate,
		ErrorTypes:               errorTypes,
		Records:                  records,
		Truncated:                len(typeTruncated) > 0,
		TypeTruncated:            typeTruncated,
		DetailFieldsAvailable:    detailFieldsAvailable,
		DetailStorageHint:        detailStorageHint,
		AvailableDetailFields:    availableDetailFields,
		ExpectedDetailSources:    expectedDetailSources,
		ReceivedDetailSources:    receivedDetailSources,
		MissingDetailSources:     missingDetailSources,
		DetailUploadWarning:      detailUploadWarning,
		ResponseCodeDistribution: responseCodeDistribution,
		ErrorTimeline:            errorTimeline,
		TopErrorMessages:         topErrorMessages,
		CategoryDistribution:     categoryDistribution,
		SourceDistribution:       sourceDistribution,
		APIDistribution:          apiDistribution,
	}
	analysis.ReportHighlights = buildErrorReportHighlights(analysis)
	setCachedErrorAnalysis(execID, cacheSignature, analysis)
	if err := saveIndexedErrorAnalysis(resultPath, cacheSignature, analysis); err != nil {
		fmt.Printf("[错误分析索引][警告] 保存执行 #%d 的索引失败: %v\n", execID, err)
	}
	return analysis, nil
}

func GetExecutionErrorOverview(execID int64) (*ErrorAnalysisOverview, error) {
	analysis, err := GetExecutionErrors(execID)
	if err != nil {
		return nil, err
	}
	return buildErrorAnalysisOverview(analysis), nil
}

// StreamExecutionLog 流式读取执行日志
func StreamExecutionLog(id int64, writer io.Writer, stopChan chan struct{}) error {
	// 获取日志路径
	var logPath string
	err := database.DB.QueryRow(
		"SELECT log_path FROM executions WHERE id = ?",
		id,
	).Scan(&logPath)
	if err != nil {
		return fmt.Errorf("查询执行记录失败: %w", err)
	}

	if logPath == "" {
		return fmt.Errorf("日志路径为空")
	}

	// 等待日志文件创建
	for {
		_, err := os.Stat(logPath)
		if err == nil {
			break
		}
		select {
		case <-stopChan:
			return nil
		case <-time.After(100 * time.Millisecond):
			continue
		}
	}

	// 打开日志文件
	file, err := os.Open(logPath)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	// 使用 bufio.Scanner 逐行读取
	scanner := bufio.NewScanner(file)
	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		if scanner.Scan() {
			line := scanner.Text()
			_, err := writer.Write([]byte(line + "\n"))
			if err != nil {
				return err
			}
		} else {
			// 检查执行是否完成
			var status string
			database.DB.QueryRow("SELECT status FROM executions WHERE id = ?", id).Scan(&status)
			if status != "running" {
				// 执行完成，再读取剩余内容
				for scanner.Scan() {
					line := scanner.Text()
					writer.Write([]byte(line + "\n"))
				}
				return nil
			}
			// 等待新内容
			time.Sleep(100 * time.Millisecond)
		}
	}
}

var runtimeDependencyPropNames = map[string]bool{
	"File.path":                       true,
	"filename":                        true,
	"scriptFile":                      true,
	"BeanShellSampler.filename":       true,
	"BeanShellPreProcessor.filename":  true,
	"BeanShellPostProcessor.filename": true,
	"BSFSampler.filename":             true,
	"BSFPreProcessor.filename":        true,
	"BSFPostProcessor.filename":       true,
	"JSR223Sampler.filename":          true,
	"JSR223PreProcessor.filename":     true,
	"JSR223PostProcessor.filename":    true,
	"JSR223Assertion.filename":        true,
	"JSR223Listener.filename":         true,
	"HTTPSampler.file.path":           true,
	"HTTPFileArg.path":                true,
}

func rewriteRuntimeDependencyProps(jmxContent string, rewriter func(propName, value string) (string, bool)) string {
	stringPropRe := regexp.MustCompile(`<stringProp name="([^"]+)">(.*?)</stringProp>`)
	return stringPropRe.ReplaceAllStringFunc(jmxContent, func(match string) string {
		submatches := stringPropRe.FindStringSubmatch(match)
		if len(submatches) < 3 {
			return match
		}

		propName := strings.TrimSpace(submatches[1])
		value := strings.TrimSpace(submatches[2])
		if !runtimeDependencyPropNames[propName] || !shouldTrackDependencyPath(value) {
			return match
		}

		rewritten, changed := rewriter(propName, value)
		if !changed {
			return match
		}

		return `<stringProp name="` + propName + `">` + rewritten + `</stringProp>`
	})
}

func absolutizeRuntimeDependencyPaths(jmxContent string, baseDir string) string {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return jmxContent
	}
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		absBaseDir = baseDir
	}

	return rewriteRuntimeDependencyProps(jmxContent, func(_ string, value string) (string, bool) {
		if filepath.IsAbs(value) {
			return "", false
		}
		return filepath.ToSlash(filepath.Join(absBaseDir, value)), true
	})
}

func replaceFileDependencyPaths(jmxContent string, remoteDataDir string, fileNames []string) string {
	fileNameMap := make(map[string]string, len(fileNames))
	fileNameSet := make(map[string]bool, len(fileNames))
	for _, name := range fileNames {
		base := filepath.Base(strings.TrimSpace(name))
		if base != "" {
			fileNameSet[base] = true
			fileNameMap[base] = base
		}
	}

	return replaceFileDependencyPathsWithMap(jmxContent, remoteDataDir, fileNameMap)
}

func replaceFileDependencyPathsWithMap(jmxContent string, remoteDataDir string, fileNameMap map[string]string) string {
	return rewriteRuntimeDependencyProps(jmxContent, func(_ string, value string) (string, bool) {
		value = strings.TrimSpace(value)
		targetName, ok := fileNameMap[value]
		if !ok {
			targetName, ok = fileNameMap[filepath.Base(value)]
		}
		if !ok || strings.TrimSpace(targetName) == "" {
			return "", false
		}
		return filepath.ToSlash(filepath.Join(remoteDataDir, targetName)), true
	})
}

func resolveRuntimeDependencySourcePath(dep string, scriptDir string, attachedLookup map[string]string) (string, error) {
	dep = strings.TrimSpace(dep)
	if dep == "" {
		return "", fmt.Errorf("依赖路径为空")
	}

	if filepath.IsAbs(dep) {
		if _, err := os.Stat(dep); err != nil {
			return "", err
		}
		return dep, nil
	}

	base := filepath.Base(dep)
	if attachedPath := strings.TrimSpace(attachedLookup[base]); attachedPath != "" {
		if _, err := os.Stat(attachedPath); err == nil {
			return attachedPath, nil
		}
	}

	candidate := filepath.Join(scriptDir, dep)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}

	return "", fmt.Errorf("文件不存在或不可读: %s", dep)
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// ExecutionSummary 执行摘要（用于对比）
type ExecutionSummary struct {
	ID           int64   `json:"id"`
	ScriptName   string  `json:"script_name"`
	Status       string  `json:"status"`
	StartTime    string  `json:"start_time"`
	Duration     int64   `json:"duration"`
	TotalSamples int64   `json:"total_samples"`
	AvgRT        float64 `json:"avg_rt"`
	TPS          float64 `json:"tps"`
	ErrorRate    float64 `json:"error_rate"`
	P90RT        float64 `json:"p90_rt"`
	P95RT        float64 `json:"p95_rt"`
	P99RT        float64 `json:"p99_rt"`
	IsBaseline   bool    `json:"is_baseline"`
}

// MetricDiff 指标差异
type MetricDiff struct {
	Metric   string  `json:"metric"` // "avg_rt", "tps", "error_rate" 等
	Label    string  `json:"label"`  // 中文标签
	Value1   float64 `json:"value1"`
	Value2   float64 `json:"value2"`
	DiffPct  float64 `json:"diff_pct"` // 变化百分比
	Improved bool    `json:"improved"` // 是否改善
	Unit     string  `json:"unit"`     // 单位
}

// ComparisonResult 对比结果
type ComparisonResult struct {
	Execution1  ExecutionSummary `json:"execution1"`
	Execution2  ExecutionSummary `json:"execution2"`
	Differences []MetricDiff     `json:"differences"`
}

// SetBaseline 设置基准线
func SetBaseline(executionID int64) error {
	// 查询该执行记录，获取 script_id
	var scriptID int64
	err := database.DB.QueryRow("SELECT script_id FROM executions WHERE id = ?", executionID).Scan(&scriptID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("执行记录不存在")
		}
		return fmt.Errorf("查询执行记录失败: %w", err)
	}

	// 使用事务保证原子性
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 将该 script_id 下所有执行的 is_baseline 置为 0
	_, err = tx.Exec("UPDATE executions SET is_baseline = 0 WHERE script_id = ?", scriptID)
	if err != nil {
		return fmt.Errorf("重置基准线失败: %w", err)
	}

	// 将该执行的 is_baseline 置为 1
	_, err = tx.Exec("UPDATE executions SET is_baseline = 1 WHERE id = ?", executionID)
	if err != nil {
		return fmt.Errorf("设置基准线失败: %w", err)
	}

	return tx.Commit()
}

// UnsetBaseline 取消基准线
func UnsetBaseline(executionID int64) error {
	_, err := database.DB.Exec("UPDATE executions SET is_baseline = 0 WHERE id = ?", executionID)
	if err != nil {
		return fmt.Errorf("取消基准线失败: %w", err)
	}
	return nil
}

// CompareExecutions 对比两次执行
func CompareExecutions(id1, id2 int64) (*ComparisonResult, error) {
	// 查询两个执行记录
	exec1, err := GetExecution(id1)
	if err != nil {
		return nil, fmt.Errorf("获取执行记录1失败: %w", err)
	}
	exec2, err := GetExecution(id2)
	if err != nil {
		return nil, fmt.Errorf("获取执行记录2失败: %w", err)
	}

	// 解析 summary_data
	var summary1, summary2 map[string]interface{}
	if exec1.SummaryData != "" {
		json.Unmarshal([]byte(exec1.SummaryData), &summary1)
	}
	if exec2.SummaryData != "" {
		json.Unmarshal([]byte(exec2.SummaryData), &summary2)
	}

	// 构建 ExecutionSummary
	es1 := buildExecutionSummary(exec1, summary1)
	es2 := buildExecutionSummary(exec2, summary2)

	// 计算差异
	diffs := calculateDifferences(summary1, summary2)

	return &ComparisonResult{
		Execution1:  es1,
		Execution2:  es2,
		Differences: diffs,
	}, nil
}

// buildExecutionSummary 从执行记录和 summary_data 构建 ExecutionSummary
func buildExecutionSummary(exec *model.Execution, summary map[string]interface{}) ExecutionSummary {
	es := ExecutionSummary{
		ID:         exec.ID,
		ScriptName: exec.ScriptName,
		Status:     exec.Status,
		StartTime:  exec.StartTime,
		Duration:   exec.Duration,
		IsBaseline: exec.IsBaseline,
	}

	if summary != nil {
		// 提取指标
		if v, ok := summary["total_samples"].(float64); ok {
			es.TotalSamples = int64(v)
		}
		if v, ok := summary["avg_response_time"].(float64); ok {
			es.AvgRT = v
		}
		if v, ok := summary["transaction_tps"].(float64); ok {
			es.TPS = v
		} else if v, ok := summary["primary_throughput"].(float64); ok {
			es.TPS = v
		} else if v, ok := summary["throughput"].(float64); ok {
			es.TPS = v
		}
		if v, ok := summary["error_rate"].(float64); ok {
			es.ErrorRate = v
		}
		if v, ok := summary["p90"].(float64); ok {
			es.P90RT = v
		}
		if v, ok := summary["p95"].(float64); ok {
			es.P95RT = v
		}
		if v, ok := summary["p99"].(float64); ok {
			es.P99RT = v
		}
	}

	return es
}

// calculateDifferences 计算指标差异
func calculateDifferences(summary1, summary2 map[string]interface{}) []MetricDiff {
	var diffs []MetricDiff

	// 定义要对比的指标
	metrics := []struct {
		key            string
		label          string
		unit           string
		higherIsBetter bool
	}{
		{"avg_response_time", "平均响应时间", "ms", false},
		{"transaction_tps", "TPS（事务/s）", "tps", true},
		{"request_rate", "请求次数（次/秒）", "req/s", true},
		{"error_rate", "错误率", "%", false},
		{"p90", "P90响应时间", "ms", false},
		{"p95", "P95响应时间", "ms", false},
		{"p99", "P99响应时间", "ms", false},
	}

	for _, m := range metrics {
		value1 := getFloatValue(summary1, m.key)
		value2 := getFloatValue(summary2, m.key)

		diffPct := 0.0
		if value1 != 0 {
			diffPct = ((value2 - value1) / value1) * 100
		}

		// 判断是否改善
		// 对于 TPS: 越高越好，value2 > value1 → improved=true
		// 对于 avg_rt/error_rate: 越低越好，value2 < value1 → improved=true
		improved := false
		if m.higherIsBetter {
			improved = value2 > value1
		} else {
			improved = value2 < value1
		}

		diffs = append(diffs, MetricDiff{
			Metric:   m.key,
			Label:    m.label,
			Value1:   value1,
			Value2:   value2,
			DiffPct:  diffPct,
			Improved: improved,
			Unit:     m.unit,
		})
	}

	return diffs
}

// getFloatValue 从 map 中获取 float64 值
func getFloatValue(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// GetBaselineForScript 获取脚本的基准线执行记录
func GetBaselineForScript(scriptID int64) (*model.Execution, error) {
	var e model.Execution
	var endTime, summaryData, remarks sql.NullString
	var duration sql.NullInt64
	var isBaseline, saveHTTPDetails, includeMaster, splitCSV int
	err := database.DB.QueryRow(
		"SELECT id, script_id, script_name, slave_ids, status, start_time, end_time, duration, remarks, save_http_details, include_master, split_csv, result_path, report_path, summary_data, log_path, is_baseline, created_at FROM executions WHERE script_id = ? AND is_baseline = 1",
		scriptID,
	).Scan(
		&e.ID, &e.ScriptID, &e.ScriptName, &e.SlaveIDs, &e.Status,
		&e.StartTime, &endTime, &duration, &remarks, &saveHTTPDetails, &includeMaster, &splitCSV, &e.ResultPath, &e.ReportPath,
		&summaryData, &e.LogPath, &isBaseline, &e.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("该脚本没有设置基准线")
		}
		return nil, fmt.Errorf("查询基准线失败: %w", err)
	}

	if endTime.Valid {
		e.EndTime = endTime.String
	}
	if duration.Valid {
		e.Duration = duration.Int64
	}
	if remarks.Valid {
		e.Remarks = remarks.String
	}
	if summaryData.Valid {
		e.SummaryData = summaryData.String
	}
	e.SaveHTTPDetails = saveHTTPDetails == 1
	e.IncludeMaster = includeMaster == 1
	e.SplitCSV = splitCSV == 1
	e.IsBaseline = isBaseline == 1

	return &e, nil
}
