package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/model"
)

type executionEnvironmentReport struct {
	Role                  string   `json:"role"`
	Node                  string   `json:"node"`
	AgentVersion          string   `json:"agent_version,omitempty"`
	JMeterPath            string   `json:"jmeter_path,omitempty"`
	JMeterVersion         string   `json:"jmeter_version,omitempty"`
	JMeterVersionRaw      string   `json:"jmeter_version_raw,omitempty"`
	PluginJars            []string `json:"plugin_jars,omitempty"`
	PluginFingerprint     string   `json:"plugin_fingerprint,omitempty"`
	PropertiesLines       []string `json:"properties_lines,omitempty"`
	PropertiesFingerprint string   `json:"properties_fingerprint,omitempty"`
	Warnings              []string `json:"warnings,omitempty"`
}

type executionEnvironmentDifference struct {
	Node     string   `json:"node"`
	Category string   `json:"category"`
	Severity string   `json:"severity,omitempty"`
	Summary  string   `json:"summary"`
	Baseline string   `json:"baseline,omitempty"`
	Current  string   `json:"current,omitempty"`
	Added    []string `json:"added,omitempty"`
	Missing  []string `json:"missing,omitempty"`
}

type executionEnvironmentSnapshot struct {
	CheckedAt   string                           `json:"checked_at"`
	Baseline    executionEnvironmentReport       `json:"baseline"`
	Nodes       []executionEnvironmentReport     `json:"nodes"`
	Warnings    []string                         `json:"warnings"`
	Differences []executionEnvironmentDifference `json:"differences,omitempty"`
}

func hashStrings(values []string) string {
	sum := sha256.Sum256([]byte(strings.Join(values, "\n")))
	return hex.EncodeToString(sum[:])
}

func normalizePropertiesContent(data []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	normalized := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		separator := strings.IndexAny(line, "=:")
		if separator == -1 {
			normalized = append(normalized, line)
			continue
		}

		key := strings.TrimSpace(line[:separator])
		value := strings.TrimSpace(line[separator+1:])
		normalized = append(normalized, key+"="+value)
	}

	sort.Strings(normalized)
	return strings.Join(normalized, "\n")
}

func collectNormalizedPropertiesLines(paths []string) []string {
	lines := make([]string, 0)
	for _, path := range paths {
		data, err := os.ReadFile(path)
		prefix := filepath.Base(path)
		if err != nil {
			lines = append(lines, prefix+":__missing__")
			continue
		}
		normalized := normalizePropertiesContent(data)
		if normalized == "" {
			lines = append(lines, prefix+":__empty__")
			continue
		}
		for _, line := range strings.Split(normalized, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lines = append(lines, prefix+":"+line)
		}
	}
	sort.Strings(lines)
	return lines
}

func fingerprintFiles(paths []string) string {
	parts := make([]string, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			parts = append(parts, path+"|missing")
			continue
		}
		content := data
		if strings.HasSuffix(path, ".properties") {
			content = []byte(normalizePropertiesContent(data))
		}
		sum := sha256.Sum256(content)
		parts = append(parts, path+"|"+hex.EncodeToString(sum[:]))
	}
	return hashStrings(parts)
}

func parseJMeterVersion(output string) string {
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, field := range strings.Fields(line) {
			if strings.Count(field, ".") >= 1 && strings.ContainsAny(field, "0123456789") {
				return strings.Trim(field, " ,")
			}
		}
	}
	return ""
}

func buildJMeterVersionCandidates(executable string) []string {
	candidates := []string{executable}
	base := filepath.Base(executable)
	if strings.Contains(base, "jmeter-server") {
		fallback := filepath.Join(filepath.Dir(executable), strings.Replace(base, "jmeter-server", "jmeter", 1))
		if fallback != executable {
			candidates = append(candidates, fallback)
		}
	}
	return dedupeStrings(candidates)
}

func detectLocalJMeterVersion(executable string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, candidate := range buildJMeterVersionCandidates(executable) {
		for _, arg := range []string{"--version", "-v"} {
			cmd := exec.CommandContext(ctx, candidate, arg)
			output, err := cmd.CombinedOutput()
			if err != nil && len(output) == 0 {
				continue
			}
			if version := parseJMeterVersion(string(output)); version != "" {
				return version, nil
			}
		}
	}

	return "", nil
}

func collectLocalExecutionEnvironment() executionEnvironmentReport {
	report := executionEnvironmentReport{
		Role:     "master-local",
		Node:     "master-local",
		Warnings: []string{},
	}

	path := strings.TrimSpace(config.GlobalConfig.JMeter.Path)
	if path == "" {
		path = "jmeter"
	}

	executable, err := exec.LookPath(path)
	if err != nil {
		report.Warnings = append(report.Warnings, "未找到 Master 本地 JMeter 可执行文件")
		return report
	}

	report.JMeterPath = executable
	if version, versionErr := detectLocalJMeterVersion(executable); versionErr == nil {
		report.JMeterVersion = version
	} else {
		report.Warnings = append(report.Warnings, "读取 Master 本地 JMeter 版本失败: "+versionErr.Error())
	}

	jmeterHome := filepath.Dir(filepath.Dir(executable))
	pluginFiles, _ := filepath.Glob(filepath.Join(jmeterHome, "lib", "ext", "*.jar"))
	sort.Strings(pluginFiles)
	pluginNames := make([]string, 0, len(pluginFiles))
	for _, pluginFile := range pluginFiles {
		pluginNames = append(pluginNames, filepath.Base(pluginFile))
	}
	report.PluginJars = pluginNames
	report.PluginFingerprint = hashStrings(pluginNames)
	propertyPaths := []string{
		filepath.Join(jmeterHome, "bin", "jmeter.properties"),
		filepath.Join(jmeterHome, "bin", "user.properties"),
	}
	report.PropertiesLines = collectNormalizedPropertiesLines(propertyPaths)
	report.PropertiesFingerprint = fingerprintFiles(propertyPaths)

	return report
}

func collectSlaveExecutionEnvironment(slave model.Slave) (executionEnvironmentReport, error) {
	client := NewAgentClient(slave.Host, slave.AgentPort, slave.AgentToken)
	report, err := client.GetEnvironmentReport()
	if err != nil {
		return executionEnvironmentReport{}, err
	}

	nodeName := strings.TrimSpace(slave.Host)
	if strings.TrimSpace(slave.Name) != "" {
		nodeName = fmt.Sprintf("%s (%s)", slave.Host, slave.Name)
	}

	return executionEnvironmentReport{
		Role:                  "slave",
		Node:                  nodeName,
		AgentVersion:          report.AgentVersion,
		JMeterPath:            report.JMeterPath,
		JMeterVersion:         report.JMeterVersion,
		JMeterVersionRaw:      report.JMeterVersionRaw,
		PluginJars:            report.PluginJars,
		PluginFingerprint:     report.PluginFingerprint,
		PropertiesLines:       report.PropertiesLines,
		PropertiesFingerprint: report.PropertiesFingerprint,
		Warnings:              report.Warnings,
	}, nil
}

func diffStringSets(base, current []string) ([]string, []string) {
	baseSet := make(map[string]struct{}, len(base))
	currentSet := make(map[string]struct{}, len(current))
	for _, item := range base {
		baseSet[item] = struct{}{}
	}
	for _, item := range current {
		currentSet[item] = struct{}{}
	}

	missing := make([]string, 0)
	added := make([]string, 0)
	for _, item := range base {
		if _, exists := currentSet[item]; !exists {
			missing = append(missing, item)
		}
	}
	for _, item := range current {
		if _, exists := baseSet[item]; !exists {
			added = append(added, item)
		}
	}
	return dedupeStrings(added), dedupeStrings(missing)
}

var ignorablePropertyKeys = map[string]struct{}{
	"jmeter.properties:remote_hosts":             {},
	"user.properties:remote_hosts":               {},
	"jmeter.properties:java.rmi.server.hostname": {},
	"user.properties:java.rmi.server.hostname":   {},
	"jmeter.properties:server.rmi.localport":     {},
	"user.properties:server.rmi.localport":       {},
	"jmeter.properties:server.rmi.port":          {},
	"user.properties:server.rmi.port":            {},
	"jmeter.properties:server.rmi.ssl.disable":   {},
	"user.properties:server.rmi.ssl.disable":     {},
}

func extractPropertyLineKey(line string) string {
	separator := strings.Index(line, "=")
	if separator == -1 {
		return strings.TrimSpace(line)
	}
	return strings.TrimSpace(line[:separator])
}

func splitPropertyDifferences(lines []string) (important []string, ignorable []string) {
	for _, line := range lines {
		key := extractPropertyLineKey(line)
		if _, ok := ignorablePropertyKeys[key]; ok {
			ignorable = append(ignorable, line)
			continue
		}
		important = append(important, line)
	}
	return dedupeStrings(important), dedupeStrings(ignorable)
}

func buildEnvironmentDifferenceWarnings(differences []executionEnvironmentDifference) []string {
	warnings := make([]string, 0, len(differences))
	for _, diff := range differences {
		if diff.Severity == "info" {
			continue
		}
		warnings = append(warnings, diff.Summary)
	}
	return warnings
}

func compareExecutionEnvironments(base executionEnvironmentReport, others []executionEnvironmentReport) []executionEnvironmentDifference {
	differences := make([]executionEnvironmentDifference, 0)

	for _, report := range others {
		if strings.TrimSpace(report.JMeterVersion) == "" {
			differences = append(differences, executionEnvironmentDifference{
				Node:     report.Node,
				Category: "jmeter_version",
				Summary:  fmt.Sprintf("%s 未上报 JMeter 版本", report.Node),
				Baseline: base.JMeterVersion,
			})
		} else if strings.TrimSpace(base.JMeterVersion) != "" && report.JMeterVersion != base.JMeterVersion {
			differences = append(differences, executionEnvironmentDifference{
				Node:     report.Node,
				Category: "jmeter_version",
				Summary:  fmt.Sprintf("%s 的 JMeter 版本为 %s，基线节点为 %s", report.Node, report.JMeterVersion, base.JMeterVersion),
				Baseline: base.JMeterVersion,
				Current:  report.JMeterVersion,
			})
		}

		if base.PluginFingerprint != "" && report.PluginFingerprint != "" && report.PluginFingerprint != base.PluginFingerprint {
			added, missing := diffStringSets(base.PluginJars, report.PluginJars)
			differences = append(differences, executionEnvironmentDifference{
				Node:     report.Node,
				Category: "plugins",
				Severity: "warning",
				Summary:  fmt.Sprintf("%s 的插件清单与基线节点不一致", report.Node),
				Added:    added,
				Missing:  missing,
			})
		}

		if base.PropertiesFingerprint != "" && report.PropertiesFingerprint != "" && report.PropertiesFingerprint != base.PropertiesFingerprint {
			added, missing := diffStringSets(base.PropertiesLines, report.PropertiesLines)
			importantAdded, ignorableAdded := splitPropertyDifferences(added)
			importantMissing, ignorableMissing := splitPropertyDifferences(missing)
			if len(importantAdded) > 0 || len(importantMissing) > 0 {
				differences = append(differences, executionEnvironmentDifference{
					Node:     report.Node,
					Category: "properties",
					Severity: "warning",
					Summary:  fmt.Sprintf("%s 的 jmeter/user.properties 存在关键差异，需要关注执行一致性", report.Node),
					Added:    importantAdded,
					Missing:  importantMissing,
				})
			}
			if len(ignorableAdded) > 0 || len(ignorableMissing) > 0 {
				differences = append(differences, executionEnvironmentDifference{
					Node:     report.Node,
					Category: "properties_runtime",
					Severity: "info",
					Summary:  fmt.Sprintf("%s 存在节点级 properties 差异（通常可忽略）", report.Node),
					Added:    ignorableAdded,
					Missing:  ignorableMissing,
				})
			}
		}

		if strings.TrimSpace(base.AgentVersion) != "" && strings.TrimSpace(report.AgentVersion) != "" && report.AgentVersion != base.AgentVersion {
			differences = append(differences, executionEnvironmentDifference{
				Node:     report.Node,
				Category: "agent_version",
				Severity: "warning",
				Summary:  fmt.Sprintf("%s 的 Agent 版本为 %s，基线节点为 %s", report.Node, report.AgentVersion, base.AgentVersion),
				Baseline: base.AgentVersion,
				Current:  report.AgentVersion,
			})
		}
	}

	return differences
}

func validateExecutionEnvironmentReport(report executionEnvironmentReport) []string {
	issues := make([]string, 0)
	node := report.Node
	if strings.TrimSpace(node) == "" {
		node = report.Role
	}
	if strings.TrimSpace(report.JMeterPath) == "" {
		issues = append(issues, fmt.Sprintf("%s 未上报 JMeter 可执行文件路径", node))
	}
	if strings.TrimSpace(report.JMeterVersion) == "" {
		issues = append(issues, fmt.Sprintf("%s 未上报 JMeter 版本", node))
	}
	if strings.TrimSpace(report.PluginFingerprint) == "" {
		issues = append(issues, fmt.Sprintf("%s 未上报插件指纹", node))
	}
	if strings.TrimSpace(report.PropertiesFingerprint) == "" {
		issues = append(issues, fmt.Sprintf("%s 未上报 properties 指纹", node))
	}
	for _, warning := range report.Warnings {
		warning = strings.TrimSpace(warning)
		if warning == "" {
			continue
		}
		issues = append(issues, fmt.Sprintf("%s: %s", node, warning))
	}
	return issues
}

func validateExecutionEnvironments(slaves []model.Slave, includeMaster bool) (*executionEnvironmentSnapshot, error) {
	if len(slaves) == 0 {
		return nil, nil
	}

	reports := make([]executionEnvironmentReport, 0, len(slaves)+1)
	var baseline executionEnvironmentReport
	if includeMaster {
		baseline = collectLocalExecutionEnvironment()
		reports = append(reports, baseline)
	}

	for _, slave := range slaves {
		report, err := collectSlaveExecutionEnvironment(slave)
		if err != nil {
			return nil, fmt.Errorf("获取 %s(%s) 环境报告失败: %w", slave.Name, slave.Host, err)
		}
		reports = append(reports, report)
	}

	if !includeMaster && len(reports) > 0 {
		baseline = reports[0]
	}

	issues := make([]string, 0)
	for _, report := range reports {
		issues = append(issues, validateExecutionEnvironmentReport(report)...)
	}
	if len(reports) > 1 {
		differences := compareExecutionEnvironments(baseline, reports[1:])
		issues = append(issues, buildEnvironmentDifferenceWarnings(differences)...)
		snapshot := &executionEnvironmentSnapshot{
			CheckedAt:   time.Now().Format("2006-01-02 15:04:05"),
			Baseline:    baseline,
			Nodes:       reports,
			Warnings:    dedupeStrings(issues),
			Differences: differences,
		}
		return snapshot, nil
	}

	snapshot := &executionEnvironmentSnapshot{
		CheckedAt: time.Now().Format("2006-01-02 15:04:05"),
		Baseline:  baseline,
		Nodes:     reports,
		Warnings:  dedupeStrings(issues),
	}
	return snapshot, nil
}

func saveExecutionEnvironmentSnapshot(resultDir string, snapshot *executionEnvironmentSnapshot) error {
	if snapshot == nil {
		return nil
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(resultDir, "environment-report.json"), data, 0644)
}

func loadExecutionEnvironmentSnapshot(resultDir string) (*executionEnvironmentSnapshot, error) {
	data, err := os.ReadFile(filepath.Join(resultDir, "environment-report.json"))
	if err != nil {
		return nil, err
	}
	var snapshot executionEnvironmentSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}
