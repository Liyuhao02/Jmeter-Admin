package router

import (
	"embed"
	"io/fs"
	"net/http"

	"jmeter-admin/config"
	"jmeter-admin/internal/handler"

	"github.com/gin-gonic/gin"
)

func SetupRouter(frontendFS embed.FS) *gin.Engine {
	r := gin.Default()

	// CORS 中间件
	r.Use(corsMiddleware())

	// API 路由组
	api := r.Group("/api")
	{
		// Scripts 路由
		scripts := api.Group("/scripts")
		{
			scripts.GET("", handler.ListScripts)
			scripts.GET("/stats", handler.GetScriptStats)
			scripts.POST("", handler.CreateScript)
			scripts.GET("/:id", handler.GetScript)
			scripts.GET("/:id/dependencies", handler.GetScriptDependencies)
			scripts.PUT("/:id", handler.UpdateScript)
			scripts.DELETE("/:id", handler.DeleteScript)
			scripts.GET("/:id/download", handler.DownloadScript)
			scripts.GET("/:id/content", handler.GetScriptContent)
			scripts.PUT("/:id/content", handler.SaveScriptContent)
			scripts.POST("/:id/files", handler.UploadFiles)
			scripts.DELETE("/:id/files/:fileId", handler.DeleteFile)
			scripts.GET("/:id/versions", handler.GetScriptVersions)
			scripts.GET("/:id/versions/:versionId", handler.GetScriptVersionContent)
			scripts.POST("/:id/versions/:versionId/restore", handler.RestoreScriptVersion)
		}

		// Slaves 路由
		slaves := api.Group("/slaves")
		{
			slaves.GET("", handler.ListSlaves)
			slaves.POST("", handler.CreateSlave)
			slaves.PUT("/:id", handler.UpdateSlave)
			slaves.DELETE("/:id", handler.DeleteSlave)
			slaves.POST("/:id/check", handler.CheckSlave)
			slaves.GET("/heartbeat-status", handler.GetHeartbeatStatus)
		}

		// Executions 路由
		executions := api.Group("/executions")
		{
			executions.GET("", handler.ListExecutions)
			executions.GET("/stats", handler.GetExecutionStats)   // 统计汇总API
			executions.GET("/compare", handler.CompareExecutions) // 执行对比API（放在 /:id 之前）
			executions.GET("/callback-probe", handler.CallbackProbe)
			executions.POST("", handler.CreateExecution)
			executions.GET("/:id", handler.GetExecution)
			executions.GET("/:id/live-metrics", handler.GetExecutionLiveMetrics)
			executions.GET("/:id/node-metrics", handler.GetExecutionNodeMetrics)
			executions.PUT("/:id/baseline", handler.SetBaseline)
			executions.DELETE("/:id", handler.DeleteExecution)
			executions.POST("/:id/stop", handler.StopExecution)
			executions.GET("/:id/log", handler.GetExecutionLog)
			executions.GET("/:id/errors", handler.GetExecutionErrors)
			executions.POST("/:id/error-details/upload", handler.UploadExecutionErrorDetails)
			executions.GET("/:id/download/jtl", handler.DownloadResultFile)
			executions.GET("/:id/download/report", handler.DownloadReport)
			executions.GET("/:id/download/errors", handler.ExportErrors)
			executions.GET("/:id/download/all", handler.DownloadAll)
		}

		// Config 路由（系统配置相关）
		configGroup := api.Group("/config")
		{
			configGroup.GET("/network-interfaces", handler.GetNetworkInterfaces)
			configGroup.GET("/master-hostname", handler.GetMasterHostname)
			configGroup.PUT("/master-hostname", handler.UpdateMasterHostname)
		}
	}

	// JMeter 报告静态文件服务
	r.Static("/reports", config.GlobalConfig.Dirs.Results)

	// 获取前端文件系统
	fsys, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		panic(err)
	}
	frontendFileSystem := http.FS(fsys)

	// 前端静态文件服务（嵌入的 web/dist）
	r.GET("/assets/*filepath", func(c *gin.Context) {
		c.FileFromFS("/assets/"+c.Param("filepath"), frontendFileSystem)
	})

	// 其他所有路由 fallback 到前端 index.html（支持 Vue Router history 模式）
	r.NoRoute(func(c *gin.Context) {
		// API 路由不应该 fallback
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.AbortWithStatus(404)
			return
		}
		// 尝试打开请求的文件
		file, err := fsys.Open(c.Request.URL.Path)
		if err != nil {
			// 文件不存在，返回 index.html 支持 Vue Router history 模式
			c.FileFromFS("/", frontendFileSystem)
			return
		}
		file.Close()
		// 文件存在，正常服务
		c.FileFromFS(c.Request.URL.Path, frontendFileSystem)
	})

	return r
}

// CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
