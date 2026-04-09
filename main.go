package main

import (
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"jmeter-admin/config"
	"jmeter-admin/internal/database"
	"jmeter-admin/internal/router"
	"jmeter-admin/internal/service"
)

//go:embed all:web/dist
var frontendFS embed.FS

func init() {
	// 设置全局时区为中国时区 (Asia/Shanghai, UTC+8)
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("加载时区失败: %v", err)
	}
	time.Local = loc
}

func main() {
	// 加载配置
	if err := config.LoadConfig(""); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建必要目录
	if err := createDirectories(); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer database.CloseDB()

	// 清理陈旧的执行记录（服务重启时将 running 状态置为 failed）
	if err := service.CleanupStaleExecutions(); err != nil {
		log.Printf("警告：清理陈旧执行记录失败: %v", err)
	}

	// 启动 Slave 心跳检测
	heartbeatInterval := time.Duration(config.GlobalConfig.Slave.HeartbeatInterval) * time.Second
	if heartbeatInterval <= 0 {
		heartbeatInterval = 30 * time.Second
	}
	service.StartHeartbeat(heartbeatInterval)

	// 设置路由
	r := router.SetupRouter(frontendFS)

	// 启动服务
	addr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	log.Printf("JMeter Admin 服务启动在 http://localhost%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

func createDirectories() error {
	dirs := []string{
		config.GlobalConfig.Dirs.Data,
		config.GlobalConfig.Dirs.Uploads,
		config.GlobalConfig.Dirs.Results,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}
	}

	return nil
}
