package config

import (
	"errors"
	"fmt"
	"os"

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

	// Executor 特有配置
	AllowedDispatchers []string `yaml:"allowed_dispatchers,omitempty"`
	AllowedScripts     []string `yaml:"allowed_scripts,omitempty"`
}

// ServiceConfig 定义服务配置
type ServiceConfig struct {
	Bots          []BotConfig `yaml:"bots"`
	RobotGroupID  string      `yaml:"robot_group_id"`
	TaskWhiteList []string    `yaml:"task_whitelist"`
	UserWhiteList []string    `yaml:"user_whitelist"`
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
