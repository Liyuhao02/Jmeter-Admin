package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
)

const version = "1.0.0"

var agentStartTime = time.Now()

type SystemStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	Disk    DiskStats    `json:"disk"`
	Network NetworkStats `json:"network"`
}

type CPUStats struct {
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

type MemoryStats struct {
	TotalMB uint64  `json:"total_mb"`
	UsedMB  uint64  `json:"used_mb"`
	Percent float64 `json:"percent"`
}

type DiskStats struct {
	TotalMB uint64  `json:"total_mb"`
	UsedMB  uint64  `json:"used_mb"`
	Percent float64 `json:"percent"`
}

type NetworkStats struct {
	Connections int `json:"connections"`
}

func collectSystemStats() *SystemStats {
	stats := &SystemStats{}

	// CPU - 使用 cpu.Percent(500ms, false) 快速采集
	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err == nil && len(cpuPercent) > 0 {
		stats.CPU.Percent = math.Round(cpuPercent[0]*10) / 10 // 保留1位小数
	}
	cpuCount, _ := cpu.Counts(false)
	stats.CPU.Count = cpuCount

	// Memory
	vmem, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory.TotalMB = vmem.Total / (1024 * 1024)
		stats.Memory.UsedMB = vmem.Used / (1024 * 1024)
		stats.Memory.Percent = math.Round(vmem.UsedPercent*10) / 10
	}

	// Disk - 根分区
	diskUsage, err := disk.Usage("/")
	if err == nil {
		stats.Disk.TotalMB = diskUsage.Total / (1024 * 1024)
		stats.Disk.UsedMB = diskUsage.Used / (1024 * 1024)
		stats.Disk.Percent = math.Round(diskUsage.UsedPercent*10) / 10
	}

	// Network connections
	conns, err := psnet.Connections("all")
	if err == nil {
		stats.Network.Connections = len(conns)
	}

	return stats
}

type Server struct {
	dataDir    string
	token      string
	jmeterPath string
	mux        *http.ServeMux
}

func NewServer(dataDir, token, jmeterPath string) *Server {
	s := &Server{
		dataDir:    dataDir,
		token:      token,
		jmeterPath: strings.TrimSpace(jmeterPath),
		mux:        http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/api/files/upload", s.authMiddleware(s.handleUpload))
	s.mux.HandleFunc("/api/files/", s.authMiddleware(s.handleFileOperations))
	s.mux.HandleFunc("/api/network/check-callback", s.authMiddleware(s.handleCheckCallback))
	s.mux.HandleFunc("/api/environment/report", s.authMiddleware(s.handleEnvironmentReport))
}

func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != s.token {
				s.writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	response := map[string]interface{}{
		"status":         "ok",
		"version":        version,
		"data_dir":       s.dataDir,
		"uptime_seconds": int64(time.Since(agentStartTime).Seconds()),
		"sys_stats":      collectSystemStats(),
	}
	s.writeJSON(w, http.StatusOK, response)
}

type callbackCheckRequest struct {
	URL string `json:"url"`
}

type callbackCheckResponse struct {
	Reachable  bool   `json:"reachable"`
	StatusCode int    `json:"status_code"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type environmentReport struct {
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

func (s *Server) handleCheckCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req callbackCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	targetURL := strings.TrimSpace(req.URL)
	if targetURL == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing callback url"})
		return
	}

	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	httpReq, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid callback url"})
		return
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		s.writeJSON(w, http.StatusOK, callbackCheckResponse{
			Reachable: false,
			LatencyMS: time.Since(start).Milliseconds(),
			Error:     err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	s.writeJSON(w, http.StatusOK, callbackCheckResponse{
		Reachable:  resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		LatencyMS:  time.Since(start).Milliseconds(),
	})
}

func fingerprintStrings(values []string) string {
	hash := sha256.Sum256([]byte(strings.Join(values, "\n")))
	return hex.EncodeToString(hash[:])
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

func collectFileFingerprint(paths []string) string {
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
	return fingerprintStrings(parts)
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

func (s *Server) resolveJMeterExecutable() (string, error) {
	path := s.jmeterPath
	if path == "" {
		if envPath := strings.TrimSpace(os.Getenv("JMETER_PATH")); envPath != "" {
			path = envPath
		} else {
			path = "jmeter"
		}
	}
	return exec.LookPath(path)
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

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func detectJMeterVersion(executable string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rawOutputs := make([]string, 0, 2)
	for _, candidate := range buildJMeterVersionCandidates(executable) {
		for _, arg := range []string{"--version", "-v"} {
			cmd := exec.CommandContext(ctx, candidate, arg)
			output, err := cmd.CombinedOutput()
			raw := strings.TrimSpace(string(output))
			if raw != "" {
				rawOutputs = append(rawOutputs, candidate+" "+arg+":\n"+raw)
			}
			if err != nil && raw == "" {
				continue
			}
			if version := parseJMeterVersion(raw); version != "" {
				return version, strings.Join(rawOutputs, "\n\n"), nil
			}
		}
	}

	return "", strings.Join(rawOutputs, "\n\n"), nil
}

func collectEnvironmentReport(jmeterPath string) environmentReport {
	report := environmentReport{
		AgentVersion: version,
		JMeterPath:   strings.TrimSpace(jmeterPath),
		PluginJars:   []string{},
		Warnings:     []string{},
	}

	executable, err := func() (string, error) {
		if strings.TrimSpace(jmeterPath) != "" {
			return exec.LookPath(jmeterPath)
		}
		if envPath := strings.TrimSpace(os.Getenv("JMETER_PATH")); envPath != "" {
			return exec.LookPath(envPath)
		}
		return exec.LookPath("jmeter")
	}()
	if err != nil {
		report.Warnings = append(report.Warnings, "未找到 JMeter 可执行文件")
		return report
	}

	report.JMeterPath = executable
	report.JMeterHome = filepath.Dir(filepath.Dir(executable))

	versionText, rawVersion, versionErr := detectJMeterVersion(executable)
	report.JMeterVersion = versionText
	report.JMeterVersionRaw = rawVersion
	if versionErr != nil {
		report.Warnings = append(report.Warnings, "读取 JMeter 版本失败: "+versionErr.Error())
	}

	pluginFiles, _ := filepath.Glob(filepath.Join(report.JMeterHome, "lib", "ext", "*.jar"))
	sort.Strings(pluginFiles)
	pluginNames := make([]string, 0, len(pluginFiles))
	for _, path := range pluginFiles {
		pluginNames = append(pluginNames, filepath.Base(path))
	}
	report.PluginJars = pluginNames
	report.PluginFingerprint = fingerprintStrings(pluginNames)

	props := []string{
		filepath.Join(report.JMeterHome, "bin", "jmeter.properties"),
		filepath.Join(report.JMeterHome, "bin", "user.properties"),
	}
	report.PropertiesLines = collectNormalizedPropertiesLines(props)
	report.PropertiesFingerprint = collectFileFingerprint(props)
	return report
}

func (s *Server) handleEnvironmentReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	s.writeJSON(w, http.StatusOK, collectEnvironmentReport(s.jmeterPath))
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to parse form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	originalFilename := header.Filename
	targetName := r.FormValue("target_name")
	if targetName == "" {
		targetName = originalFilename
	}

	if !s.isValidFilename(targetName) {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid filename"})
		return
	}

	filePath := filepath.Join(s.dataDir, targetName)

	dst, err := os.Create(filePath)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create file: " + err.Error()})
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save file: " + err.Error()})
		return
	}

	log.Printf("[Agent] POST /api/files/upload - %s (%d bytes)", targetName, size)

	response := map[string]interface{}{
		"filename": targetName,
		"size":     size,
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleFileOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/files/")

	if r.Method == http.MethodDelete {
		if path == "batch" {
			s.handleBatchDelete(w, r)
		} else {
			s.handleSingleDelete(w, r, path)
		}
		return
	}

	s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func (s *Server) handleSingleDelete(w http.ResponseWriter, r *http.Request, filename string) {
	if !s.isValidFilename(filename) {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid filename"})
		return
	}

	filePath := filepath.Join(s.dataDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}

	if err := os.Remove(filePath); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete file: " + err.Error()})
		return
	}

	log.Printf("[Agent] DELETE /api/files/%s - deleted", filename)
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prefix    string   `json:"prefix"`
		Filenames []string `json:"filenames"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	var filesToDelete []string

	if len(req.Filenames) > 0 {
		for _, filename := range req.Filenames {
			if s.isValidFilename(filename) {
				filesToDelete = append(filesToDelete, filename)
			}
		}
	} else if req.Prefix != "" {
		entries, err := os.ReadDir(s.dataDir)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read directory"})
			return
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), req.Prefix) {
				filesToDelete = append(filesToDelete, entry.Name())
			}
		}
	}

	deleted := 0
	for _, filename := range filesToDelete {
		filePath := filepath.Join(s.dataDir, filename)
		if err := os.Remove(filePath); err == nil {
			deleted++
		}
	}

	log.Printf("[Agent] DELETE /api/files/batch - deleted %d files", deleted)
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"deleted": deleted,
	})
}

func (s *Server) isValidFilename(filename string) bool {
	if filename == "" {
		return false
	}
	if strings.Contains(filename, "..") {
		return false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}
	return true
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) logRequest(method, path, filename string, size int64) {
	if filename != "" {
		log.Printf("[Agent] %s %s - %s (%d bytes)", method, path, filename, size)
	} else {
		log.Printf("[Agent] %s %s", method, path)
	}
}

func (s *Server) formatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
