package handler

import (
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"jmeter-admin/internal/model"
	"jmeter-admin/internal/service"
)

// 文件上传限制常量
const (
	maxSingleFileSize = 100 << 20 // 单文件最大 100MB
	maxTotalFileSize  = 500 << 20 // 总计最大 500MB
)

// sanitizeFileName 清理文件名，防止路径穿越攻击
func sanitizeFileName(name string) string {
	// 使用 filepath.Base 去除路径分隔符
	name = filepath.Base(name)
	// 检查是否为当前目录或上级目录
	if name == "." || name == ".." {
		return "unnamed"
	}
	// 确保文件名不为空
	if name == "" {
		return "unnamed"
	}
	return name
}

// ListScripts GET /api/scripts?page=1&page_size=10&keyword=xxx
func ListScripts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	scripts, total, err := service.ListScripts(page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PageSuccess(total, scripts))
}

// CreateScript POST /api/scripts (form-data: name, description, file)
func CreateScript(c *gin.Context) {
	name := c.PostForm("name")
	description := c.PostForm("description")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("请上传一个 .jmx 脚本文件"))
		return
	}

	if file.Size > maxSingleFileSize {
		c.JSON(http.StatusBadRequest, model.Error("脚本文件超过 100MB 限制"))
		return
	}

	safeFileName := sanitizeFileName(file.Filename)
	if !strings.HasSuffix(strings.ToLower(safeFileName), ".jmx") {
		c.JSON(http.StatusBadRequest, model.Error("只支持上传一个 .jmx 脚本文件"))
		return
	}

	if name == "" {
		name = strings.TrimSuffix(safeFileName, filepath.Ext(safeFileName))
		if name == "" {
			c.JSON(http.StatusBadRequest, model.Error("无法从文件名生成脚本名称"))
			return
		}
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无法读取上传的脚本文件"))
		return
	}
	fileData, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("读取脚本文件失败"))
		return
	}

	// 创建脚本记录
	script, err := service.CreateScript(name, description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	if _, err := service.UploadScriptFile(script.ID, safeFileName, fileData); err != nil {
		_ = service.DeleteScript(script.ID)
		c.JSON(http.StatusInternalServerError, model.Error("创建脚本失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(script))
}

// DownloadScript GET /api/scripts/:id/download
func DownloadScript(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	filePath, fileName, err := service.GetScriptDownloadInfo(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	c.FileAttachment(filePath, fileName)
}

// GetScript GET /api/scripts/:id
func GetScript(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	script, err := service.GetScript(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	// 获取关联的文件列表
	files, err := service.GetScriptFiles(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"script": script,
		"files":  files,
	}))
}

// UpdateScript PUT /api/scripts/:id (JSON: name, description)
func UpdateScript(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("请求参数无效: "+err.Error()))
		return
	}

	if err := service.UpdateScript(id, req.Name, req.Description); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

// DeleteScript DELETE /api/scripts/:id
func DeleteScript(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	if err := service.DeleteScript(id); err != nil {
		c.JSON(http.StatusInternalServerError, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

// GetScriptContent GET /api/scripts/:id/content
func GetScriptContent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	content, err := service.GetScriptContent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(gin.H{
		"content": content,
	}))
}

// SaveScriptContent PUT /api/scripts/:id/content (JSON: {content: "xml字符串"})
func SaveScriptContent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Error("请求参数无效: "+err.Error()))
		return
	}

	if err := service.SaveScriptContent(id, req.Content); err != nil {
		c.JSON(http.StatusBadRequest, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

// UploadFiles POST /api/scripts/:id/files (form-data: files[])
func UploadFiles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	// 设置请求体大小限制
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxTotalFileSize)

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无法解析上传的文件: "+err.Error()))
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, model.Error("没有上传文件"))
		return
	}

	var uploadedFiles []*model.ScriptFile
	totalSize := int64(0)

	for _, file := range files {
		// 检查单文件大小
		if file.Size > maxSingleFileSize {
			c.JSON(http.StatusBadRequest, model.Error("文件 '"+file.Filename+"' 超过 100MB 限制"))
			return
		}

		// 检查总大小
		totalSize += file.Size
		if totalSize > maxTotalFileSize {
			c.JSON(http.StatusBadRequest, model.Error("上传文件总大小超过 500MB 限制"))
			return
		}

		f, err := file.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}

		// 清理文件名，防止路径穿越
		safeFileName := sanitizeFileName(file.Filename)

		scriptFile, err := service.UploadScriptFile(id, safeFileName, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.Error("上传文件失败: "+err.Error()))
			return
		}
		uploadedFiles = append(uploadedFiles, scriptFile)
	}

	c.JSON(http.StatusOK, model.Success(uploadedFiles))
}

// DeleteFile DELETE /api/scripts/:id/files/:fileId
// 支持按文件ID或文件名删除
func DeleteFile(c *gin.Context) {
	scriptID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.Error("无效的脚本ID"))
		return
	}

	fileIdentifier := c.Param("fileId")
	if fileIdentifier == "" {
		c.JSON(http.StatusBadRequest, model.Error("文件标识不能为空"))
		return
	}

	// 使用支持ID和文件名的删除方法
	if err := service.DeleteScriptFileByIdentifier(scriptID, fileIdentifier); err != nil {
		c.JSON(http.StatusNotFound, model.Error(err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
