package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Frontend FrontendConfig `yaml:"frontend"`
	JMeter   JMeterConfig   `yaml:"jmeter"`
	Slave    SlaveConfig    `yaml:"slave"`
	Dirs     DirsConfig     `yaml:"dirs"`
}

type ServerConfig struct {
	Port int `yaml:"port"` // HTTP 服务监听端口
}

type FrontendConfig struct {
	Port int `yaml:"port"` // 前端开发服务器端口（仅开发模式）
}

type JMeterConfig struct {
	Path           string `yaml:"path"`            // JMeter 可执行文件路径
	MasterHostname string `yaml:"master_hostname"` // RMI 回调 IP，多网卡时必填
}

type SlaveConfig struct {
	HeartbeatInterval int `yaml:"heartbeat_interval"` // 心跳检测间隔（秒）
}

type DirsConfig struct {
	Data    string `yaml:"data"`
	Uploads string `yaml:"uploads"`
	Results string `yaml:"results"`
}

var GlobalConfig Config

func LoadConfig(configPath string) error {
	// 设置默认值
	GlobalConfig = Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Frontend: FrontendConfig{
			Port: 3000,
		},
		JMeter: JMeterConfig{
			Path: "jmeter",
		},
		Slave: SlaveConfig{
			HeartbeatInterval: 30,
		},
		Dirs: DirsConfig{
			Data:    "./data",
			Uploads: "./uploads",
			Results: "./results",
		},
	}

	// 如果配置文件存在，则加载
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 配置文件不存在，使用默认值并创建默认配置文件
			return createDefaultConfig(configPath)
		}
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := yaml.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

func createDefaultConfig(configPath string) error {
	data, err := yaml.Marshal(&GlobalConfig)
	if err != nil {
		return fmt.Errorf("生成默认配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入默认配置文件失败: %w", err)
	}

	return nil
}

// SaveConfig 保存当前配置到文件
func SaveConfig(configPath string) error {
	if configPath == "" {
		configPath = "config.yaml"
	}
	data, err := yaml.Marshal(&GlobalConfig)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}
