package config

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
