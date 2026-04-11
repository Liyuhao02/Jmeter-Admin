package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
)

type scriptDependencyScan struct {
	CSVFiles            []string
	FileDependencies    []string
	PluginDependencies  []string
	AttachedFiles       []string
	MissingDependencies []string
	Warnings            []string
}

func parseExecutionSummaryMap(summaryData string) map[string]interface{} {
	if strings.TrimSpace(summaryData) == "" {
		return map[string]interface{}{}
	}
	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(summaryData), &summary); err != nil {
		return map[string]interface{}{}
	}
	return summary
}

func summaryFloatValue(summary map[string]interface{}, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := summary[key]; ok {
			switch typed := value.(type) {
			case float64:
				return typed
			case float32:
				return float64(typed)
			case int:
				return float64(typed)
			case int64:
				return float64(typed)
			case json.Number:
				if parsed, err := typed.Float64(); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func summaryIntValue(summary map[string]interface{}, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := summary[key]; ok {
			switch typed := value.(type) {
			case float64:
				return int64(typed)
			case float32:
				return int64(typed)
			case int:
				return int64(typed)
			case int64:
				return typed
			case json.Number:
				if parsed, err := typed.Int64(); err == nil {
					return parsed
				}
			}
		}
	}
	return 0
}

func summaryStringValue(summary map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := summary[key]; ok {
			if text, ok := value.(string); ok {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func enrichExecutionForDisplay(execution *model.Execution, includeDiagnostics bool) {
	if execution == nil {
		return
	}

	summary := parseExecutionSummaryMap(execution.SummaryData)
	totalRequests := summaryIntValue(summary, "request_samples", "total_samples")
	successRequests := summaryIntValue(summary, "request_success_samples", "success_samples")
	errorRequests := summaryIntValue(summary, "request_error_samples", "error_samples")
	errorRate := summaryFloatValue(summary, "error_rate")

	execution.ProcessStatus = execution.Status
	execution.DisplayStatus = execution.Status
	execution.ResultStatus = "unknown"
	execution.StatusTone = "info"
	execution.StatusReason = "等待执行结果"

	switch execution.Status {
	case "running":
		execution.DisplayStatus = "running"
		execution.ResultStatus = "in_progress"
		execution.StatusTone = "info"
		execution.StatusReason = "压测进行中，核心指标以实时数据为准"
	case "failed":
		execution.DisplayStatus = "process_failed"
		execution.ResultStatus = "unknown"
		execution.StatusTone = "danger"
		execution.StatusReason = "JMeter 进程、结果合并或报告生成出现异常"
	case "stopped":
		execution.DisplayStatus = "stopped"
		execution.ResultStatus = "stopped"
		execution.StatusTone = "warning"
		execution.StatusReason = "任务已被手动停止，结果可能只包含部分样本"
	default:
		switch {
		case totalRequests <= 0:
			execution.DisplayStatus = "completed_no_samples"
			execution.ResultStatus = "unknown"
			execution.StatusTone = "warning"
			execution.StatusReason = "执行已完成，但没有解析到有效请求样本"
		case errorRequests == 0:
			execution.DisplayStatus = "completed_success"
			execution.ResultStatus = "healthy"
			execution.StatusTone = "success"
			execution.StatusReason = "执行完成，请求全部成功"
		case successRequests == 0:
			execution.DisplayStatus = "completed_all_failed"
			execution.ResultStatus = "all_fail"
			execution.StatusTone = "danger"
			execution.StatusReason = "执行完成，但请求全部失败"
		default:
			execution.DisplayStatus = "completed_with_errors"
			execution.ResultStatus = "partial_fail"
			execution.StatusTone = "warning"
			execution.StatusReason = fmt.Sprintf("执行完成，失败请求占比 %.2f%%", errorRate)
		}
	}

	if includeDiagnostics {
		execution.Diagnostics = buildExecutionDiagnostics(execution, summary)
	}
}

func executionFileStatus(label, path string) model.ExecutionFileStatus {
	status := model.ExecutionFileStatus{Label: label, Path: path}
	if strings.TrimSpace(path) == "" {
		return status
	}
	info, err := os.Stat(path)
	if err != nil {
		return status
	}
	status.Exists = true
	status.Size = info.Size()
	return status
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func getExecutionSlaveHosts(execution *model.Execution) []string {
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
	query := fmt.Sprintf("SELECT host, name FROM slaves WHERE id IN (%s) ORDER BY id ASC", strings.Join(placeholders, ","))
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var hosts []string
	for rows.Next() {
		var host, name string
		if err := rows.Scan(&host, &name); err != nil {
			continue
		}
		display := host
		if strings.TrimSpace(name) != "" {
			display = fmt.Sprintf("%s (%s)", host, name)
		}
		hosts = append(hosts, display)
	}
	return hosts
}

func getExecutionScriptPath(scriptID int64) string {
	var path string
	if err := database.DB.QueryRow("SELECT file_path FROM scripts WHERE id = ?", scriptID).Scan(&path); err != nil {
		return ""
	}
	return path
}

func getAttachedScriptFileNames(scriptID int64) []string {
	rows, err := database.DB.Query("SELECT file_name FROM script_files WHERE script_id = ?", scriptID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}
	return dedupeStrings(names)
}

func shouldTrackDependencyPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if strings.Contains(path, "${") || strings.HasPrefix(path, "__P(") {
		return false
	}
	lower := strings.ToLower(path)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return false
	}
	return true
}

func inspectJMXDependencies(scriptPath string, attachedFiles []string, distributed bool, splitCSV bool) scriptDependencyScan {
	scan := scriptDependencyScan{AttachedFiles: dedupeStrings(attachedFiles)}
	if strings.TrimSpace(scriptPath) == "" {
		scan.Warnings = append(scan.Warnings, "脚本主文件路径为空，无法分析依赖")
		return scan
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		scan.Warnings = append(scan.Warnings, "读取脚本内容失败，无法分析依赖")
		return scan
	}

	jmxContent := string(content)
	csvRefs := extractCSVDataSetReferences(jmxContent)
	scan.CSVFiles = dedupeStrings(extractCSVDataSetFiles(jmxContent))

	attachedSet := make(map[string]bool)
	for _, name := range scan.AttachedFiles {
		attachedSet[filepath.Base(name)] = true
	}

	csvBaseNameOwners := make(map[string]map[string]bool)
	csvHeaderConfig := make(map[string][]csvDataSetReference)
	for _, ref := range csvRefs {
		if _, ok := csvBaseNameOwners[ref.BaseName]; !ok {
			csvBaseNameOwners[ref.BaseName] = make(map[string]bool)
		}
		csvBaseNameOwners[ref.BaseName][ref.Filename] = true
		csvHeaderConfig[ref.Filename] = append(csvHeaderConfig[ref.Filename], ref)
	}

	stringPropRe := regexp.MustCompile(`<stringProp name="([^"]+)">(.*?)</stringProp>`)
	fileProps := map[string]bool{
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

	csvBaseSet := make(map[string]bool)
	for _, csvFile := range scan.CSVFiles {
		csvBaseSet[filepath.Base(csvFile)] = true
	}

	var genericFiles []string
	for _, match := range stringPropRe.FindAllStringSubmatch(jmxContent, -1) {
		if len(match) < 3 {
			continue
		}
		propName := strings.TrimSpace(match[1])
		value := strings.TrimSpace(match[2])
		if !fileProps[propName] || !shouldTrackDependencyPath(value) {
			continue
		}
		base := filepath.Base(value)
		if csvBaseSet[base] {
			continue
		}
		genericFiles = append(genericFiles, value)
	}
	scan.FileDependencies = dedupeStrings(genericFiles)

	pluginPatterns := map[string]string{
		"kg.apc.":             "JMeter Plugins (kg.apc / jp@gc)",
		"UltimateThreadGroup": "JMeter Plugins - Ultimate Thread Group",
		"jp@gc":               "JMeter Plugins (jp@gc)",
	}
	var pluginDeps []string
	for pattern, label := range pluginPatterns {
		if strings.Contains(jmxContent, pattern) {
			pluginDeps = append(pluginDeps, label)
		}
	}
	scan.PluginDependencies = dedupeStrings(pluginDeps)

	var missing []string
	for _, dep := range append(append([]string{}, scan.CSVFiles...), scan.FileDependencies...) {
		if filepath.IsAbs(dep) {
			if _, err := os.Stat(dep); err != nil {
				missing = append(missing, dep)
			}
			continue
		}
		if !attachedSet[filepath.Base(dep)] {
			missing = append(missing, dep)
		}
	}
	scan.MissingDependencies = dedupeStrings(missing)

	if distributed && len(scan.CSVFiles) > 0 && !splitCSV {
		scan.Warnings = append(scan.Warnings, "检测到 CSV 数据文件，但当前未开启 CSV 数据分片，分布式执行可能重复消费同一批数据。")
	}
	if distributed && splitCSV && len(scan.CSVFiles) > 1 {
		scan.Warnings = append(scan.Warnings, "检测到多个 CSV 输入源。若这些文件在业务上需要按行配对，请确认数据量和拆分策略一致，否则不同节点拿到的数据区间可能不再严格对齐。")
	}
	for baseName, owners := range csvBaseNameOwners {
		if len(owners) > 1 {
			scan.Warnings = append(scan.Warnings, fmt.Sprintf("检测到同名 CSV 文件 %s 来自多个路径，系统会在执行期自动重命名分发以避免冲突。", baseName))
		}
	}
	for filename, refs := range csvHeaderConfig {
		if _, consistent := hasConsistentCSVHeaderConfig(refs); !consistent {
			scan.Warnings = append(scan.Warnings, fmt.Sprintf("CSV 文件 %s 在多个 CSVDataSet 中 ignoreFirstLine 配置不一致，分片时会回退为整文件分发。", filename))
		}
	}
	if distributed && len(scan.FileDependencies) > 0 {
		scan.Warnings = append(scan.Warnings, "检测到脚本引用本地文件或脚本文件，系统会在执行期自动分发到 Slave 节点；若存在同名文件，执行期会使用唯一文件名避免冲突。")
	}
	if distributed && len(scan.PluginDependencies) > 0 {
		scan.Warnings = append(scan.Warnings, "检测到第三方 JMeter 插件组件，请确认 Master 与所有 Slave 已安装同版本插件。")
	}
	if len(scan.MissingDependencies) > 0 {
		scan.Warnings = append(scan.Warnings, "脚本引用的部分依赖文件未在当前脚本关联文件中发现，执行时可能直接失败。")
	}

	return scan
}

func readDetailSourceName(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			break
		}
		if source, ok := payload["source"].(string); ok {
			return strings.TrimSpace(source)
		}
		break
	}
	return ""
}

func buildExecutionDiagnostics(execution *model.Execution, summary map[string]interface{}) *model.ExecutionDiagnostics {
	diagnostics := &model.ExecutionDiagnostics{}
	if execution == nil || strings.TrimSpace(execution.ResultPath) == "" {
		return diagnostics
	}

	resultDir := filepath.Dir(execution.ResultPath)
	slaveHosts := getExecutionSlaveHosts(execution)
	localResultPath := filepath.Join(resultDir, "result-local.jtl")
	remoteResultPath := filepath.Join(resultDir, "result-remote.jtl")
	splitCSV := executionFileStatus("CSV 分片目录", filepath.Join(resultDir, "csv-data")).Exists

	mode := "local"
	includeMaster := false
	if len(slaveHosts) > 0 {
		mode = "distributed"
		includeMaster = executionFileStatus("本地结果", localResultPath).Exists || executionFileStatus("本地运行脚本", filepath.Join(resultDir, "runtime-local.jmx")).Exists || executionFileStatus("本地明细脚本", filepath.Join(resultDir, "runtime-local-with-details.jmx")).Exists
		if includeMaster {
			mode = "distributed_with_master"
		}
	}

	resultFiles := []model.ExecutionFileStatus{
		executionFileStatus("最终结果", execution.ResultPath),
	}
	if len(slaveHosts) > 0 {
		resultFiles = append(resultFiles, executionFileStatus("远端结果", remoteResultPath))
		if includeMaster {
			resultFiles = append(resultFiles, executionFileStatus("本地结果", localResultPath))
		}
	}

	runtimeCandidates := []model.ExecutionFileStatus{
		executionFileStatus("运行脚本", filepath.Join(resultDir, "runtime.jmx")),
		executionFileStatus("本地运行脚本", filepath.Join(resultDir, "runtime-local.jmx")),
		executionFileStatus("远端运行脚本", filepath.Join(resultDir, "runtime-remote.jmx")),
		executionFileStatus("带明细脚本", filepath.Join(resultDir, "runtime-with-details.jmx")),
		executionFileStatus("本地明细脚本", filepath.Join(resultDir, "runtime-local-with-details.jmx")),
		executionFileStatus("远端明细脚本", filepath.Join(resultDir, "runtime-remote-with-details.jmx")),
	}
	runtimeScripts := make([]model.ExecutionFileStatus, 0, len(runtimeCandidates))
	saveHTTPDetails := false
	for _, item := range runtimeCandidates {
		if item.Exists {
			runtimeScripts = append(runtimeScripts, item)
			if strings.Contains(item.Label, "明细") {
				saveHTTPDetails = true
			}
		}
	}

	localDetailFile := executionFileStatus("本地错误明细", filepath.Join(resultDir, "error-details.ndjson"))
	remoteDetailDir := filepath.Join(resultDir, "error-details")
	remoteDetailFiles := make([]model.ExecutionFileStatus, 0)
	receivedSources := make([]string, 0)
	if localDetailFile.Exists {
		saveHTTPDetails = true
		if source := readDetailSourceName(localDetailFile.Path); source != "" {
			receivedSources = append(receivedSources, source)
		} else {
			receivedSources = append(receivedSources, "master-local")
		}
	}
	if entries, err := os.ReadDir(remoteDetailDir); err == nil {
		saveHTTPDetails = true
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ndjson") {
				continue
			}
			filePath := filepath.Join(remoteDetailDir, entry.Name())
			status := executionFileStatus(strings.TrimSuffix(entry.Name(), ".ndjson"), filePath)
			remoteDetailFiles = append(remoteDetailFiles, status)
			if source := readDetailSourceName(filePath); source != "" {
				receivedSources = append(receivedSources, source)
			} else {
				receivedSources = append(receivedSources, strings.TrimSuffix(entry.Name(), ".ndjson"))
			}
		}
	}
	receivedSources = dedupeStrings(receivedSources)

	expectedSources := getExecutionExpectedDetailSources(execution)
	if includeMaster && saveHTTPDetails {
		expectedSources = append(expectedSources, "master-local")
	}
	expectedSources = dedupeStrings(expectedSources)

	missingSources := make([]string, 0)
	for _, expected := range expectedSources {
		matched := false
		for _, received := range receivedSources {
			if sourceMatchesExpected(expected, received) {
				matched = true
				break
			}
		}
		if !matched {
			missingSources = append(missingSources, expected)
		}
	}

	detailState := "disabled"
	switch {
	case !saveHTTPDetails:
		detailState = "disabled"
	case len(expectedSources) == 0:
		if localDetailFile.Exists && localDetailFile.Size > 0 {
			detailState = "local_captured"
		} else if execution.Status == "running" {
			detailState = "pending"
		} else {
			detailState = "missing"
		}
	case len(missingSources) == 0 && (len(receivedSources) > 0 || (localDetailFile.Exists && localDetailFile.Size > 0)):
		detailState = "complete"
	case len(receivedSources) > 0 || (localDetailFile.Exists && localDetailFile.Size > 0):
		detailState = "partial"
	case execution.Status == "running":
		detailState = "pending"
	default:
		detailState = "missing"
	}

	scriptPath := getExecutionScriptPath(execution.ScriptID)
	attachedFiles := getAttachedScriptFileNames(execution.ScriptID)
	dependencyScan := inspectJMXDependencies(scriptPath, attachedFiles, len(slaveHosts) > 0, splitCSV)

	warnings := append([]string{}, dependencyScan.Warnings...)
	if envSnapshot, err := loadExecutionEnvironmentSnapshot(resultDir); err == nil && envSnapshot != nil {
		if strings.TrimSpace(envSnapshot.Baseline.Node) != "" {
			warnings = append(warnings, fmt.Sprintf("环境一致性基线节点: %s", envSnapshot.Baseline.Node))
		}
		warnings = append(warnings, envSnapshot.Warnings...)
	}
	if len(slaveHosts) > 0 && len(missingSources) > 0 && execution.Status != "running" {
		warnings = append(warnings, "部分分布式节点未回传错误明细，错误分析可能并不完整。")
	}
	if len(slaveHosts) > 0 {
		finalResult := resultFiles[0]
		if !finalResult.Exists {
			warnings = append(warnings, "最终合并结果文件尚未生成，当前结果可能仍来自本地或远端单边文件。")
		}
	}

	diagnostics.Mode = mode
	diagnostics.IncludeMaster = includeMaster
	diagnostics.SlaveCount = len(slaveHosts)
	diagnostics.SlaveHosts = slaveHosts
	diagnostics.ResultFiles = resultFiles
	diagnostics.RuntimeScripts = runtimeScripts
	diagnostics.ResultMergeReady = resultFiles[0].Exists && resultFiles[0].Size > 0
	diagnostics.SaveHTTPDetails = saveHTTPDetails
	diagnostics.DetailState = detailState
	diagnostics.SplitCSV = splitCSV
	if localDetailFile.Exists || saveHTTPDetails {
		diagnostics.DetailLocalFile = &localDetailFile
	}
	diagnostics.DetailRemoteFiles = remoteDetailFiles
	diagnostics.ExpectedDetailSources = expectedSources
	diagnostics.ReceivedDetailSources = receivedSources
	diagnostics.MissingDetailSources = missingSources
	diagnostics.CSVDependencies = dependencyScan.CSVFiles
	diagnostics.FileDependencies = dependencyScan.FileDependencies
	diagnostics.PluginDependencies = dependencyScan.PluginDependencies
	diagnostics.MissingDependencies = dependencyScan.MissingDependencies
	diagnostics.Warnings = dedupeStrings(warnings)

	_ = summary
	return diagnostics
}
