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

// TaskDetail 任务详细配置
type TaskDetail struct {
	Script string   `yaml:"script"`   // 脚本路径
	Params []string `yaml:"params"`   // 参数列表
}

// ExecutorTask 定义执行机器人的单个任务配置
type ExecutorTask struct {
	// Name 唯一任务名称（全局唯一，用于 dispatcher 发送和 executor 认领）
	// 格式建议：{server}-{env}-{operation}，如 "potato-dev-restart"
	Name string `yaml:"name"`
	// DisplayName 可读性好的任务显示名称（用于用户反馈）
	// 如 "土豆开发服重启"
	DisplayName string `yaml:"display_name,omitempty"`
	// Keywords 用于精确匹配用户输入中的关键实体（如服务器名）
	// 用户输入中包含任意一个 keyword 即命中该任务
	Keywords []string `yaml:"keywords"`
	// Names 操作名候选列表，支持别名
	// 用户输入经 keywords 过滤后的剩余文本会与这些名称进行匹配
	Names []string `yaml:"names"`
	// Script 对应脚本路径
	Script string `yaml:"script"`
}

// BotConfig 定义单个机器人的配置
type BotConfig struct {
	Name      string `yaml:"name"`
	AppID     string `yaml:"app_id"`
	AppSecret string `yaml:"app_secret"`
	Role      string `yaml:"role"` // "dispatcher" | "executor"

	// Dispatcher 特有配置
	AllowedUsers    []string          `yaml:"allowed_users,omitempty"` // 允许的用户列表（仅 dispatcher 角色）
	GroupProjectMap map[string]string `yaml:"group_project_map,omitempty"` // 群组 ID 到项目名的映射（仅 dispatcher 角色），如 "oc_xxx": "土豆"

	// Executor 特有配置
	AllowedDispatchers []string          `yaml:"allowed_dispatchers,omitempty"` // 允许的 dispatcher app_id 列表
	RoutingKeywords    []string          `yaml:"routing_keywords,omitempty"`    // Executor 路由关键词（粗粒度模糊匹配）
	Description        string            `yaml:"description,omitempty"`          // Executor 描述（用于日志输出）
	LegacyTasks        []ExecutorTask    `yaml:"legacy_tasks,omitempty"`        // 旧格式任务列表（待迁移，将被 Tasks 字段替代）
	Tasks              map[string]TaskDetail `yaml:"tasks,omitempty"`           // 任务详细配置（参数化任务，新格式）
	TaskScripts        map[string]string `yaml:"task_scripts,omitempty"`        // 旧版任务映射（兼容exact模式）
	PollInterval       string            `yaml:"poll_interval,omitempty"`      // 轮询间隔（如 "1s", "500ms"，默认 "1s"）
	MaxTasks           int               `yaml:"max_tasks,omitempty"`          // 最大并发任务数，0 或负数表示不限制，默认 0

	// 语义匹配配置
	SemanticMatch *SemanticMatchConfig `yaml:"semantic_match,omitempty"`
}

// SemanticMatchConfig 语义匹配配置
type SemanticMatchConfig struct {
	Enabled   bool    `yaml:"enabled"`
	Threshold float64 `yaml:"threshold"` // 相似度阈值，默认 0.7
	Provider  string `yaml:"provider"`  // "glm" | "openai"
	Model     string `yaml:"model"`     // 模型名（glm: embedding-3, openai: text-embedding-ada-002）
	APIKey    string `yaml:"api_key"`   // API Key
	BaseURL   string `yaml:"base_url"`  // 自定义 API 地址（可选）
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
