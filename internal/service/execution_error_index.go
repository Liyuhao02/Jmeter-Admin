package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type errorAnalysisIndexFile struct {
	Signature string         `json:"signature"`
	UpdatedAt string         `json:"updated_at"`
	Analysis  *ErrorAnalysis `json:"analysis"`
}

func errorAnalysisIndexPath(resultPath string) string {
	return filepath.Join(filepath.Dir(resultPath), "error-analysis.index.json")
}

func loadIndexedErrorAnalysis(resultPath, signature string) (*ErrorAnalysis, bool) {
	if strings.TrimSpace(resultPath) == "" || strings.TrimSpace(signature) == "" {
		return nil, false
	}

	data, err := os.ReadFile(errorAnalysisIndexPath(resultPath))
	if err != nil {
		return nil, false
	}

	var payload errorAnalysisIndexFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}
	if payload.Signature != signature || payload.Analysis == nil {
		return nil, false
	}
	return payload.Analysis, true
}

func saveIndexedErrorAnalysis(resultPath, signature string, analysis *ErrorAnalysis) error {
	if strings.TrimSpace(resultPath) == "" || strings.TrimSpace(signature) == "" || analysis == nil {
		return nil
	}

	payload := errorAnalysisIndexFile{
		Signature: signature,
		UpdatedAt: time.Now().Format("2006-01-02 15:04:05"),
		Analysis:  analysis,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(errorAnalysisIndexPath(resultPath), data, 0644)
}

func refreshExecutionErrorAnalysisIndex(execID int64, logWriter io.Writer) {
	if _, err := GetExecutionErrors(execID); err != nil && logWriter != nil {
		fmt.Fprintf(logWriter, "[执行 #%d] 刷新错误分析索引失败: %v\n", execID, err)
	}
}

func startExecutionErrorAnalysisIndexer(execID int64, stop <-chan struct{}, logWriter io.Writer) {
	refreshExecutionErrorAnalysisIndex(execID, logWriter)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			refreshExecutionErrorAnalysisIndex(execID, logWriter)
			return
		case <-ticker.C:
			refreshExecutionErrorAnalysisIndex(execID, logWriter)
		}
	}
}
