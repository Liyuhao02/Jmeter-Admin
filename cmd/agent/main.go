package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"jmeter-admin/internal/agent"
)

func main() {
	var port string
	var dataDir string
	var token string
	var jmeterPath string

	flag.StringVar(&port, "port", "8089", "监听端口")
	flag.StringVar(&dataDir, "data-dir", "./csv-data", "CSV 文件存放目录")
	flag.StringVar(&token, "token", "", "鉴权 token（可选，为空则不校验）")
	flag.StringVar(&jmeterPath, "jmeter-path", "", "JMeter 可执行文件路径（默认读取 JMETER_PATH 或系统 PATH）")
	flag.Parse()

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("[Agent] 无法创建数据目录 %s: %v", dataDir, err)
	}

	server := agent.NewServer(dataDir, token, jmeterPath)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("[Agent] 收到退出信号，正在关闭服务...")
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%s", port)
	log.Printf("[Agent] 启动成功，监听端口 %s，数据目录: %s", port, dataDir)
	if token != "" {
		log.Println("[Agent] 已启用 token 鉴权")
	}
	log.Printf("[Agent] 健康检查: http://localhost%s/health", addr)

	if err := server.Start(addr); err != nil {
		log.Fatalf("[Agent] 服务启动失败: %v", err)
	}
}
