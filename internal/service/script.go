package service

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
)

// ListScripts 分页查询脚本列表，支持按name模糊搜索
func ListScripts(page, pageSize int, keyword string) ([]model.Script, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 构建查询条件
	var whereClause string
	var args []interface{}
	if keyword != "" {
		whereClause = " WHERE s.name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}

	// 查询总数
	var total int64
	countQuery := "SELECT COUNT(*) FROM scripts s" + whereClause
	err := database.DB.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("查询脚本总数失败: %w", err)
	}

	// 查询列表
	query := `
		SELECT
			s.id,
			s.name,
			s.description,
			s.file_path,
			COALESCE((
				SELECT sf.file_name
				FROM script_files sf
				WHERE sf.script_id = s.id AND sf.file_type = 'jmx'
				ORDER BY sf.created_at DESC
				LIMIT 1
			), '') AS file_name,
			(
				SELECT COUNT(*)
				FROM script_files sf_count
				WHERE sf_count.script_id = s.id
			) AS file_count,
			s.created_at,
			s.updated_at
		FROM scripts s` + whereClause + " ORDER BY s.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询脚本列表失败: %w", err)
	}
	defer rows.Close()

	var scripts []model.Script
	for rows.Next() {
		var s model.Script
		err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.FilePath, &s.FileName, &s.FileCount, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描脚本数据失败: %w", err)
		}
		scripts = append(scripts, s)
	}

	return scripts, total, nil
}

func GetScriptStats() (*model.ScriptStats, error) {
	stats := &model.ScriptStats{}

	if err := database.DB.QueryRow("SELECT COUNT(*) FROM scripts").Scan(&stats.TotalScripts); err != nil {
		return nil, fmt.Errorf("查询脚本总数失败: %w", err)
	}

	if err := database.DB.QueryRow("SELECT COUNT(*) FROM script_files").Scan(&stats.TotalFiles); err != nil {
		return nil, fmt.Errorf("查询脚本文件总数失败: %w", err)
	}

	return stats, nil
}

// CreateScript 创建脚本记录
func CreateScript(name, description string) (*model.Script, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	script := &model.Script{
		Name:        name,
		Description: description,
		FilePath:    "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := database.DB.Exec(
		"INSERT INTO scripts (name, description, file_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		script.Name, script.Description, script.FilePath, script.CreatedAt, script.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建脚本失败: %w", err)
	}

	script.ID, err = result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取脚本ID失败: %w", err)
	}

	// 创建脚本文件目录
	scriptDir := filepath.Join(config.GlobalConfig.Dirs.Uploads, fmt.Sprintf("%d", script.ID))
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return nil, fmt.Errorf("创建脚本目录失败: %w", err)
	}

	return script, nil
}

// GetScript 获取单个脚本详情
func GetScript(id int64) (*model.Script, error) {
	var script model.Script
	err := database.DB.QueryRow(
		"SELECT id, name, description, file_path, created_at, updated_at FROM scripts WHERE id = ?",
		id,
	).Scan(&script.ID, &script.Name, &script.Description, &script.FilePath, &script.CreatedAt, &script.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("脚本不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询脚本失败: %w", err)
	}

	return &script, nil
}

// InspectScriptDependencies 检查脚本依赖与分布式执行风险
func InspectScriptDependencies(id int64, distributed bool, splitCSV bool) (*model.ScriptDependencyReport, error) {
	script, err := GetScript(id)
	if err != nil {
		return nil, err
	}

	attachedFiles := getAttachedScriptFileNames(id)
	scan := inspectJMXDependencies(script.FilePath, attachedFiles, distributed, splitCSV)

	report := &model.ScriptDependencyReport{
		CSVFiles:            scan.CSVFiles,
		FileDependencies:    scan.FileDependencies,
		PluginDependencies:  scan.PluginDependencies,
		AttachedFiles:       scan.AttachedFiles,
		MissingDependencies: scan.MissingDependencies,
		Warnings:            scan.Warnings,
	}
	return report, nil
}

// GetScriptDownloadInfo 获取脚本主文件下载信息
func GetScriptDownloadInfo(id int64) (string, string, error) {
	script, err := GetScript(id)
	if err != nil {
		return "", "", err
	}

	if script.FilePath == "" {
		return "", "", fmt.Errorf("脚本没有关联的jmx文件")
	}

	if _, err := os.Stat(script.FilePath); err != nil {
		if os.IsNotExist(err) {
			return "", "", fmt.Errorf("脚本文件不存在")
		}
		return "", "", fmt.Errorf("读取脚本文件失败: %w", err)
	}

	return script.FilePath, filepath.Base(script.FilePath), nil
}

// UpdateScript 更新脚本信息
func UpdateScript(id int64, name, description string) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := database.DB.Exec(
		"UPDATE scripts SET name = ?, description = ?, updated_at = ? WHERE id = ?",
		name, description, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新脚本失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("脚本不存在")
	}

	return nil
}

// DeleteScript 删除脚本（同时删除关联文件记录和磁盘文件）
func DeleteScript(id int64) error {
	// 先获取所有关联的文件路径
	rows, err := database.DB.Query("SELECT file_path FROM script_files WHERE script_id = ?", id)
	if err != nil {
		return fmt.Errorf("查询脚本文件失败: %w", err)
	}
	defer rows.Close()

	var filePaths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return fmt.Errorf("扫描文件路径失败: %w", err)
		}
		filePaths = append(filePaths, path)
	}

	// 删除数据库记录（外键会自动删除 script_files 记录）
	result, err := database.DB.Exec("DELETE FROM scripts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除脚本失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("脚本不存在")
	}

	// 删除磁盘文件
	for _, path := range filePaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			// 记录错误但不中断
			fmt.Printf("删除文件失败 %s: %v\n", path, err)
		}
	}

	// 删除脚本目录
	scriptDir := filepath.Join(config.GlobalConfig.Dirs.Uploads, fmt.Sprintf("%d", id))
	if err := os.RemoveAll(scriptDir); err != nil {
		// 记录错误但不中断
		fmt.Printf("删除脚本目录失败 %s: %v\n", scriptDir, err)
	}

	return nil
}

// GetScriptContent 读取jmx文件内容返回字符串
func GetScriptContent(id int64) (string, error) {
	script, err := GetScript(id)
	if err != nil {
		return "", err
	}

	if script.FilePath == "" {
		return "", fmt.Errorf("脚本没有关联的jmx文件")
	}

	content, err := os.ReadFile(script.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("jmx文件不存在")
		}
		return "", fmt.Errorf("读取jmx文件失败: %w", err)
	}

	return string(content), nil
}

// SaveScriptContent 保存jmx内容到文件（先校验XML格式有效性）
func SaveScriptContent(id int64, content string) error {
	// 校验XML格式
	if err := validateXML(content); err != nil {
		return fmt.Errorf("XML格式无效: %w", err)
	}

	script, err := GetScript(id)
	if err != nil {
		return err
	}

	if script.FilePath == "" {
		return fmt.Errorf("脚本没有关联的jmx文件")
	}

	// 写入文件
	if err := os.WriteFile(script.FilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("保存jmx文件失败: %w", err)
	}

	// 更新 updated_at
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = database.DB.Exec("UPDATE scripts SET updated_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("更新脚本时间失败: %w", err)
	}

	// 自动创建版本记录
	if err := createScriptVersion(id, content, ""); err != nil {
		// 版本记录失败不影响主流程，仅记录错误
		fmt.Printf("创建脚本版本记录失败: %v\n", err)
	}

	return nil
}

// createScriptVersion 创建脚本版本记录
func createScriptVersion(scriptID int64, content, changeSummary string) error {
	// 计算内容hash
	hash := sha256.Sum256([]byte(content))
	contentHash := hex.EncodeToString(hash[:])

	// 查询最新版本的hash
	var latestHash string
	err := database.DB.QueryRow(
		"SELECT content_hash FROM script_versions WHERE script_id = ? ORDER BY version_number DESC LIMIT 1",
		scriptID,
	).Scan(&latestHash)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("查询最新版本hash失败: %w", err)
	}

	// 如果hash相同，说明内容未变，跳过创建
	if err == nil && latestHash == contentHash {
		return nil
	}

	// 获取最新版本号
	var latestVersion int
	err = database.DB.QueryRow(
		"SELECT COALESCE(MAX(version_number), 0) FROM script_versions WHERE script_id = ?",
		scriptID,
	).Scan(&latestVersion)
	if err != nil {
		return fmt.Errorf("查询最新版本号失败: %w", err)
	}

	// 自动生成change_summary
	if changeSummary == "" {
		changeSummary = fmt.Sprintf("版本 %d", latestVersion+1)
	}

	// 插入新版本
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = database.DB.Exec(
		"INSERT INTO script_versions (script_id, version_number, content, content_hash, change_summary, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		scriptID, latestVersion+1, content, contentHash, changeSummary, now,
	)
	if err != nil {
		return fmt.Errorf("插入版本记录失败: %w", err)
	}

	// 版本保留策略：超过50个版本则删除最早的
	if err := cleanupOldVersions(scriptID); err != nil {
		fmt.Printf("清理旧版本失败: %v\n", err)
	}

	return nil
}

// cleanupOldVersions 清理旧版本，保留最近50个
func cleanupOldVersions(scriptID int64) error {
	_, err := database.DB.Exec(
		"DELETE FROM script_versions WHERE script_id = ? AND id IN (SELECT id FROM script_versions WHERE script_id = ? ORDER BY version_number DESC LIMIT -1 OFFSET 50)",
		scriptID, scriptID,
	)
	if err != nil {
		return fmt.Errorf("删除旧版本失败: %w", err)
	}
	return nil
}

// GetScriptVersions 获取脚本版本列表（不包含content字段）
func GetScriptVersions(scriptID int64) ([]model.ScriptVersion, error) {
	// 检查脚本是否存在
	_, err := GetScript(scriptID)
	if err != nil {
		return nil, err
	}

	rows, err := database.DB.Query(
		"SELECT id, script_id, version_number, content_hash, change_summary, created_at FROM script_versions WHERE script_id = ? ORDER BY version_number DESC",
		scriptID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询版本列表失败: %w", err)
	}
	defer rows.Close()

	var versions []model.ScriptVersion
	for rows.Next() {
		var v model.ScriptVersion
		err := rows.Scan(&v.ID, &v.ScriptID, &v.VersionNumber, &v.ContentHash, &v.ChangeSummary, &v.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描版本数据失败: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// GetScriptVersionContent 获取指定版本的完整内容
func GetScriptVersionContent(scriptID, versionID int64) (*model.ScriptVersion, error) {
	// 检查脚本是否存在
	_, err := GetScript(scriptID)
	if err != nil {
		return nil, err
	}

	var v model.ScriptVersion
	err = database.DB.QueryRow(
		"SELECT id, script_id, version_number, content, content_hash, change_summary, created_at FROM script_versions WHERE id = ? AND script_id = ?",
		versionID, scriptID,
	).Scan(&v.ID, &v.ScriptID, &v.VersionNumber, &v.Content, &v.ContentHash, &v.ChangeSummary, &v.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("版本不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询版本内容失败: %w", err)
	}

	return &v, nil
}

// RestoreScriptVersion 回滚到指定版本
func RestoreScriptVersion(scriptID, versionID int64) error {
	// 获取指定版本的内容
	version, err := GetScriptVersionContent(scriptID, versionID)
	if err != nil {
		return err
	}

	// 保存内容（会自动创建新版本）
	changeSummary := fmt.Sprintf("从版本 %d 回滚", version.VersionNumber)
	if err := SaveScriptContent(scriptID, version.Content); err != nil {
		return fmt.Errorf("保存回滚内容失败: %w", err)
	}

	// 更新新版本的change_summary为回滚标记
	// 由于SaveScriptContent已经创建了版本，我们需要更新最新版本的change_summary
	_, err = database.DB.Exec(
		"UPDATE script_versions SET change_summary = ? WHERE script_id = ? ORDER BY version_number DESC LIMIT 1",
		changeSummary, scriptID,
	)
	if err != nil {
		// 更新失败不影响主流程
		fmt.Printf("更新回滚版本描述失败: %v\n", err)
	}

	return nil
}

// validateXML 校验XML格式有效性
func validateXML(content string) error {
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				return nil
			}
			return err
		}
		if token == nil {
			return nil
		}
	}
}

// UploadScriptFile 保存上传文件到 uploads/{scriptID}/ 目录，写入 script_files 表
func UploadScriptFile(scriptID int64, fileName string, fileData []byte) (*model.ScriptFile, error) {
	// 检查脚本是否存在
	_, err := GetScript(scriptID)
	if err != nil {
		return nil, err
	}

	// 确定文件类型
	fileType := getFileType(fileName)

	// 构建文件路径
	scriptDir := filepath.Join(config.GlobalConfig.Dirs.Uploads, fmt.Sprintf("%d", scriptID))
	filePath := filepath.Join(scriptDir, fileName)

	// 确保目录存在
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}

	// 创建数据库记录
	now := time.Now().Format("2006-01-02 15:04:05")
	scriptFile := &model.ScriptFile{
		ScriptID:  scriptID,
		FileName:  fileName,
		FilePath:  filePath,
		FileType:  fileType,
		CreatedAt: now,
		UpdatedAt: now,
	}

	result, err := database.DB.Exec(
		"INSERT INTO script_files (script_id, file_name, file_path, file_type, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		scriptFile.ScriptID, scriptFile.FileName, scriptFile.FilePath, scriptFile.FileType, scriptFile.CreatedAt, scriptFile.UpdatedAt,
	)
	if err != nil {
		// 回滚文件写入
		os.Remove(filePath)
		return nil, fmt.Errorf("保存文件记录失败: %w", err)
	}

	scriptFile.ID, err = result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取文件ID失败: %w", err)
	}

	// 如果是.jmx文件，更新 scripts 表的 file_path
	if fileType == "jmx" {
		_, err = database.DB.Exec("UPDATE scripts SET file_path = ? WHERE id = ?", filePath, scriptID)
		if err != nil {
			return nil, fmt.Errorf("更新脚本主文件路径失败: %w", err)
		}
	}

	return scriptFile, nil
}

// getFileType 根据文件名获取文件类型
func getFileType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jmx":
		return "jmx"
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".txt":
		return "txt"
	case ".properties":
		return "properties"
	case ".xml":
		return "xml"
	case ".yaml", ".yml":
		return "yaml"
	case ".jar":
		return "jar"
	default:
		return "other"
	}
}

// DeleteScriptFile 删除附加文件（磁盘+数据库）
func DeleteScriptFile(scriptID, fileID int64) error {
	// 获取文件信息
	var filePath string
	var fileType string
	err := database.DB.QueryRow(
		"SELECT file_path, file_type FROM script_files WHERE id = ? AND script_id = ?",
		fileID, scriptID,
	).Scan(&filePath, &fileType)

	if err == sql.ErrNoRows {
		return fmt.Errorf("文件不存在")
	}
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}

	// 删除数据库记录
	result, err := database.DB.Exec("DELETE FROM script_files WHERE id = ? AND script_id = ?", fileID, scriptID)
	if err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("文件不存在")
	}

	// 删除磁盘文件
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		// 记录错误但不中断
		fmt.Printf("删除文件失败 %s: %v\n", filePath, err)
	}

	// 如果是jmx文件，清空 scripts 表的 file_path
	if fileType == "jmx" {
		_, err = database.DB.Exec("UPDATE scripts SET file_path = '' WHERE id = ?", scriptID)
		if err != nil {
			fmt.Printf("清空脚本主文件路径失败: %v\n", err)
		}
	}

	return nil
}

// DeleteScriptFileByIdentifier 根据文件ID或文件名删除文件
// 先尝试按ID查找，找不到则按文件名查找
func DeleteScriptFileByIdentifier(scriptID int64, identifier string) error {
	var filePath, fileType string
	var fileID int64

	// 先尝试按 ID 查找
	err := database.DB.QueryRow(
		"SELECT id, file_path, file_type FROM script_files WHERE id = ? AND script_id = ?",
		identifier, scriptID,
	).Scan(&fileID, &filePath, &fileType)

	if err == sql.ErrNoRows {
		// 找不到，尝试按文件名查找
		err = database.DB.QueryRow(
			"SELECT id, file_path, file_type FROM script_files WHERE file_name = ? AND script_id = ?",
			identifier, scriptID,
		).Scan(&fileID, &filePath, &fileType)
	}

	if err == sql.ErrNoRows {
		return fmt.Errorf("文件不存在")
	}
	if err != nil {
		return fmt.Errorf("查询文件失败: %w", err)
	}

	// 删除数据库记录
	result, err := database.DB.Exec("DELETE FROM script_files WHERE id = ?", fileID)
	if err != nil {
		return fmt.Errorf("删除文件记录失败: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("文件不存在")
	}

	// 删除磁盘文件
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("删除文件失败 %s: %v\n", filePath, err)
	}

	// 如果是jmx文件，清空 scripts 表的 file_path
	if fileType == "jmx" {
		_, err = database.DB.Exec("UPDATE scripts SET file_path = '' WHERE id = ?", scriptID)
		if err != nil {
			fmt.Printf("清空脚本主文件路径失败: %v\n", err)
		}
	}

	return nil
}

// GetScriptFiles 获取脚本关联的所有文件列表
func GetScriptFiles(scriptID int64) ([]model.ScriptFile, error) {
	// 检查脚本是否存在
	_, err := GetScript(scriptID)
	if err != nil {
		return nil, err
	}

	rows, err := database.DB.Query(
		"SELECT id, script_id, file_name, file_path, file_type, created_at, updated_at FROM script_files WHERE script_id = ? ORDER BY created_at DESC",
		scriptID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询脚本文件失败: %w", err)
	}
	defer rows.Close()

	var files []model.ScriptFile
	for rows.Next() {
		var f model.ScriptFile
		var updatedAt sql.NullString
		err := rows.Scan(&f.ID, &f.ScriptID, &f.FileName, &f.FilePath, &f.FileType, &f.CreatedAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描文件数据失败: %w", err)
		}
		if updatedAt.Valid {
			f.UpdatedAt = updatedAt.String
		} else {
			f.UpdatedAt = f.CreatedAt
		}
		files = append(files, f)
	}

	return files, nil
}

// IsValidXML 检查字符串是否是有效的XML
func IsValidXML(content string) bool {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(content)))
	for {
		token, err := decoder.Token()
		if err != nil {
			return err.Error() == "EOF"
		}
		if token == nil {
			return true
		}
	}
}
