package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Errors
var (
	ErrConfigNotFound = errors.New("config file not found")
	ErrInvalidYAML    = errors.New("invalid YAML format")
)

// BotConfig 定义单个机器人的配置
type BotConfig struct {
	Name      string `yaml:"name"`
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	Role      string `yaml:"role"` // "dispatcher" | "executor"

	// Dispatcher 特有配置
	AllowedUsers []string `yaml:"allowed_users,omitempty"` // 允许的用户列表（仅 dispatcher 角色）

	// Executor 特有配置
	AllowedDispatchers []string          `yaml:"allowed_dispatchers,omitempty"` // 允许的 dispatcher app_id 列表
	TaskScripts        map[string]string `yaml:"task_scripts,omitempty"`      // 任务名 -> 脚本路径映射（仅 executor 角色）
	PollInterval       string            `yaml:"poll_interval,omitempty"`      // 轮询间隔（如 "1s", "500ms"，仅 executor 角色，默认 "1s"）
}

// ServiceConfig 定义服务配置
type ServiceConfig struct {
	Bots         []BotConfig `yaml:"bots"`
	RobotGroupID string      `yaml:"robot_group_id"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*ServiceConfig, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Check for empty file
	if len(data) == 0 {
		return nil, fmt.Errorf("config file is empty: %s", path)
	}

	// Parse YAML
	var cfg ServiceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAML, err)
	}

	return &cfg, nil
}

// ParsePollInterval 解析轮询间隔字符串，返回 time.Duration
// 支持格式: "1s", "500ms", "1m" 等
// 如果为空或无效，返回默认值 1 秒
func ParsePollInterval(intervalStr string) time.Duration {
	if intervalStr == "" {
		return 1 * time.Second
	}
	duration, err := time.ParseDuration(intervalStr)
	if err != nil {
		return 1 * time.Second
	}
	// 限制最小间隔为 100ms，避免过于频繁的请求
	if duration < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	return duration
}
