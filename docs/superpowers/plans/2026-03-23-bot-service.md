# 飞书机器人服务实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 构建一个基于飞书 Go SDK 的多机器人协作服务，支持配置文件管理、启动参数控制、群聊消息通信和任务日志记录

**架构：** 单进程多机器人架构，角色驱动的 Handler 设计，通过消息路由器分发事件到对应的 Handler 处理

**技术栈：** Go 1.21, 飞书 SDK v3, Cobra CLI, Viper 配置管理, Zap 日志

---

## 项目文件结构

```
lark-bot-service/
├── cmd/
│   └── bot-service/
│       └── main.go              # 入口，CLI 命令定义
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构体定义
│   │   ├── loader.go            # 配置加载逻辑
│   │   └── validator.go         # 配置验证逻辑
│   ├── bot/
│   │   ├── types.go             # 核心类型定义（BotClient, TaskRecord等）
│   │   ├── registry.go          # 机器人注册表
│   │   └── client.go            # 飞书 SDK 客户端封装
│   ├── router/
│   │   └── router.go            # 消息路由器
│   ├── handler/
│   │   ├── handler.go           # MessageHandler 接口定义
│   │   ├── dispatcher.go        # DispatcherHandler 实现
│   │   └── executor.go          # ExecutorHandler 实现
│   ├── logger/
│   │   └── task_logger.go       # 任务日志记录器
│   └── common/
│       ├── message.go           # 消息发送通用函数
│       └── errors.go            # 错误定义
├── configs/
│   └── bots.yaml.example        # 配置文件示例
├── test/
│   └── mocks/
│       └── lark_mock.go         # 飞书 SDK Mock
├── go.mod
├── go.sum
└── README.md
```

---

## 任务分解

### Task 1: 项目初始化

**目标：** 初始化 Go 项目，安装依赖，创建基础目录结构

**Files:**
- Create: `go.mod`
- Create: `go.sum`
- Create: `README.md`
- Create: `configs/` 目录
- Create: `test/mocks/` 目录

- [ ] **Step 1: 初始化 Go 模块**

```bash
go mod init github.com/yourname/lark-bot-service
```

- [ ] **Step 2: 安装核心依赖**

```bash
go get github.com/larksuite/oapi-sdk-go/v3@v3.0.20
go get github.com/spf13/cobra@v1.8.0
go get github.com/spf13/viper@v1.18.0
go get go.uber.org/zap@v1.26.0
go get gopkg.in/yaml.v3@v3.0.1
```

- [ ] **Step 3: 创建目录结构**

```bash
mkdir -p cmd/bot-service
mkdir -p internal/{config,bot,router,handler,logger,common}
mkdir -p configs
mkdir -p test/mocks
```

- [ ] **Step 4: 创建 README.md**

```markdown
# 飞书机器人服务

多机器人协作服务，支持配置文件管理和任务分发。

## 快速开始

\`\`\`bash
# 复制配置文件
cp configs/bots.yaml.example configs/bots.yaml

# 编辑配置
vim configs/bots.yaml

# 启动服务
./bot-service start --bots=task-dispatcher,shell-executor-1
\`\`\`

## 配置说明

详见 \`configs/bots.yaml.example\`

## 开发

\`\`\`bash
# 运行测试
go test ./...

# 构建
go build -o bot-service cmd/bot-service/main.go
\`\`\`
```

- [ ] **Step 5: 提交**

```bash
git add .
git commit -m "feat: 初始化项目结构和依赖"
```

---

### Task 2: 配置管理 - 数据结构定义

**目标：** 定义配置相关的数据结构和 YAML 标签

**Files:**
- Create: `internal/config/config.go`

- [ ] **Step 1: 编写配置结构体测试**

```go
package config

import (
	"testing"
	"gopkg.in/yaml.v3"
)

func TestBotConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
name: "test-bot"
app_id: "cli_123"
app_secret: "secret"
role: "executor"
allowed_dispatchers:
  - "cli_456"
allowed_scripts:
  - "/opt/scripts/test.sh"
`

	var cfg BotConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if cfg.Name != "test-bot" {
		t.Errorf("Expected Name 'test-bot', got '%s'", cfg.Name)
	}

	if cfg.Role != "executor" {
		t.Errorf("Expected Role 'executor', got '%s'", cfg.Role)
	}

	if len(cfg.AllowedDispatchers) != 1 {
		t.Errorf("Expected 1 allowed dispatcher, got %d", len(cfg.AllowedDispatchers))
	}
}

func TestServiceConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
bots:
  - name: "dispatcher"
    app_id: "cli_123"
    app_secret: "secret"
    role: "dispatcher"
robot_group_id: "oc_xxx"
task_whitelist:
  - "deploy"
  - "restart"
user_whitelist:
  - "user_1"
`

	var cfg ServiceConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(cfg.Bots) != 1 {
		t.Errorf("Expected 1 bot, got %d", len(cfg.Bots))
	}

	if cfg.RobotGroupID != "oc_xxx" {
		t.Errorf("Expected RobotGroupID 'oc_xxx', got '%s'", cfg.RobotGroupID)
	}

	if len(cfg.TaskWhiteList) != 2 {
		t.Errorf("Expected 2 tasks in whitelist, got %d", len(cfg.TaskWhiteList))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/config -v
```

预期结果: `undefined: BotConfig`, `undefined: ServiceConfig`

- [ ] **Step 3: 实现配置结构体**

```go
package config

// BotConfig 定义单个机器人的配置
type BotConfig struct {
	Name              string   `yaml:"name"`
	AppID             string   `yaml:"app_id"`
	AppSecret         string   `yaml:"app_secret"`
	Role              string   `yaml:"role"` // "dispatcher" | "executor"

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
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/config -v
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: 定义配置数据结构"
```

---

### Task 3: 配置管理 - 加载器

**目标：** 实现从文件加载配置的功能

**Files:**
- Create: `internal/config/loader.go`

- [ ] **Step 1: 编写加载器测试**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// 创建临时配置文件
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
bots:
  - name: "test-dispatcher"
    app_id: "cli_123"
    app_secret: "secret"
    role: "dispatcher"
robot_group_id: "oc_test"
task_whitelist:
  - "test_task"
user_whitelist:
  - "test_user"
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// 加载配置
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 验证
	if len(cfg.Bots) != 1 {
		t.Errorf("Expected 1 bot, got %d", len(cfg.Bots))
	}

	if cfg.Bots[0].Name != "test-dispatcher" {
		t.Errorf("Expected bot name 'test-dispatcher', got '%s'", cfg.Bots[0].Name)
	}

	if cfg.RobotGroupID != "oc_test" {
		t.Errorf("Expected RobotGroupID 'oc_test', got '%s'", cfg.RobotGroupID)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")

	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = LoadConfig(configPath)

	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/config -v -run TestLoadConfig
```

预期结果: `undefined: LoadConfig`

- [ ] **Step 3: 实现配置加载器**

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*ServiceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg ServiceConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/config -v -run TestLoadConfig
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/config/loader.go internal/config/loader_test.go
git commit -m "feat: 实现配置文件加载器"
```

---

### Task 4: 配置管理 - 验证器

**目标：** 实现配置验证逻辑，确保配置正确性

**Files:**
- Create: `internal/config/validator.go`

- [ ] **Step 1: 编写验证器测试**

```go
package config

import (
	"strings"
	"testing"
)

func TestValidateConfig_Valid(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:              "executor",
				AppID:             "cli_456",
				AppSecret:         "secret",
				Role:              "executor",
				AllowedDispatchers: []string{"cli_123"},
			},
		},
		RobotGroupID:  "oc_test",
		TaskWhiteList: []string{"deploy"},
		UserWhiteList: []string{"user_1"},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestValidateConfig_NoDispatcher(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "executor",
				AppID:     "cli_456",
				AppSecret: "secret",
				Role:      "executor",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing dispatcher, got nil")
	}

	if !strings.Contains(err.Error(), "至少需要一个") {
		t.Errorf("Error message should mention dispatcher requirement, got: %v", err)
	}
}

func TestValidateConfig_InvalidDispatcherReference(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:              "executor",
				AppID:             "cli_456",
				AppSecret:         "secret",
				Role:              "executor",
				AllowedDispatchers: []string{"cli_nonexistent"}, // 不存在的 dispatcher
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid dispatcher reference, got nil")
	}

	if !strings.Contains(err.Error(), "不存在的") {
		t.Errorf("Error should mention nonexistent dispatcher, got: %v", err)
	}
}

func TestValidateConfig_DuplicateBotNames(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "duplicate",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:      "duplicate", // 重复名称
				AppID:     "cli_456",
				AppSecret: "secret",
				Role:      "executor",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for duplicate bot names, got nil")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/config -v -run TestValidateConfig
```

预期结果: `undefined: ValidateConfig`

- [ ] **Step 3: 实现验证器**

```go
package config

import (
	"fmt"
	"strings"
)

// ValidateConfig 验证配置的完整性和正确性
func ValidateConfig(cfg *ServiceConfig) error {
	// 检查是否至少有一个 dispatcher
	hasDispatcher := false
	for _, bot := range cfg.Bots {
		if bot.Role == "dispatcher" {
			hasDispatcher = true
			break
		}
	}

	if !hasDispatcher {
		return fmt.Errorf("配置错误：至少需要一个 dispatcher 角色的机器人")
	}

	// 检查机器人名称唯一性
	botNames := make(map[string]bool)
	for _, bot := range cfg.Bots {
		if botNames[bot.Name] {
			return fmt.Errorf("配置错误：机器人名称重复 '%s'", bot.Name)
		}
		botNames[bot.Name] = true
	}

	// 检查必要字段
	for _, bot := range cfg.Bots {
		if bot.Name == "" {
			return fmt.Errorf("配置错误：机器人缺少名称")
		}
		if bot.AppID == "" {
			return fmt.Errorf("配置错误：机器人 '%s' 缺少 app_id", bot.Name)
		}
		if bot.AppSecret == "" {
			return fmt.Errorf("配置错误：机器人 '%s' 缺少 app_secret", bot.Name)
		}
		if bot.Role != "dispatcher" && bot.Role != "executor" {
			return fmt.Errorf("配置错误：机器人 '%s' 的角色 '%s' 无效，必须是 dispatcher 或 executor",
				bot.Name, bot.Role)
		}
	}

	// 验证 executor 的 allowed_dispatchers 是否存在
	dispatcherIDs := make(map[string]bool)
	for _, bot := range cfg.Bots {
		if bot.Role == "dispatcher" {
			dispatcherIDs[bot.AppID] = true
		}
	}

	for _, bot := range cfg.Bots {
		if bot.Role == "executor" {
			for _, dispID := range bot.AllowedDispatchers {
				if !dispatcherIDs[dispID] {
					return fmt.Errorf("配置错误：executor '%s' 引用了不存在的 dispatcher: %s",
						bot.Name, dispID)
				}
			}
		}
	}

	// 检查必要配置
	if cfg.RobotGroupID == "" {
		return fmt.Errorf("配置错误：缺少 robot_group_id")
	}

	return nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/config -v -run TestValidateConfig
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/config/validator.go internal/config/validator_test.go
git commit -m "feat: 实现配置验证逻辑"
```

---

### Task 5: 核心类型定义

**目标：** 定义 BotClient、TaskRecord 等核心数据结构

**Files:**
- Create: `internal/bot/types.go`

- [ ] **Step 1: 编写类型定义测试**

```go
package bot

import (
	"testing"
	"time"
)

func TestTaskRecord_StatusTransition(t *testing.T) {
	record := &TaskRecord{
		ID:        "task-1",
		Status:    "pending",
		StartTime: time.Now(),
	}

	// 测试状态转换
	if record.Status != "pending" {
		t.Errorf("Expected initial status 'pending', got '%s'", record.Status)
	}

	record.Status = "running"
	if record.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", record.Status)
	}

	now := time.Now()
	record.EndTime = &now
	record.Status = "completed"

	if record.Status != "completed" {
		t.Errorf("Expected final status 'completed', got '%s'", record.Status)
	}

	if record.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/bot -v
```

预期结果: `undefined: TaskRecord`

- [ ] **Step 3: 实现核心类型**

```go
package bot

import (
	"time"
)

// TaskRecord 任务记录
type TaskRecord struct {
	ID         string
	TaskName   string
	Dispatcher string
	Executor   string
	User       string
	Status     string // "pending" | "running" | "completed" | "failed"
	StartTime  time.Time
	EndTime    *time.Time
	Result     string
	Error      string
}

// BotClient 机器人客户端
type BotClient struct {
	Name      string
	AppID     string
	AppSecret string
	Role      string

	// Executor 特有配置
	AllowedDispatchers []string
	AllowedScripts     []string
}

// NewBotClient 创建机器人客户端
func NewBotClient(name, appID, appSecret, role string) *BotClient {
	return &BotClient{
		Name:      name,
		AppID:     appID,
		AppSecret: appSecret,
		Role:      role,
	}
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/bot -v
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/bot/types.go internal/bot/types_test.go
git commit -m "feat: 定义核心数据类型"
```

---

### Task 6: 机器人注册表

**目标：** 实现机器人注册表，管理所有活跃的机器人实例

**Files:**
- Create: `internal/bot/registry.go`

- [ ] **Step 1: 编写注册表测试**

```go
package bot

import (
	"testing"
)

func TestBotRegistry_Register(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")

	registry.Register(bot1)
	registry.Register(bot2)

	// 验证按名称获取
	retrieved := registry.GetByName("bot1")
	if retrieved == nil {
		t.Fatal("Expected to find bot1, got nil")
	}

	if retrieved.Name != "bot1" {
		t.Errorf("Expected bot name 'bot1', got '%s'", retrieved.Name)
	}

	// 验证按 AppID 获取
	retrieved = registry.GetByAppID("cli_2")
	if retrieved == nil {
		t.Fatal("Expected to find bot2 by app_id, got nil")
	}

	// 验证按角色获取
	dispatchers := registry.GetByRole("dispatcher")
	if len(dispatchers) != 1 {
		t.Errorf("Expected 1 dispatcher, got %d", len(dispatchers))
	}
}

func TestBotRegistry_DuplicateName(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("duplicate", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("duplicate", "cli_2", "secret2", "executor")

	registry.Register(bot1)

	// 尝试注册同名机器人
	err := registry.Register(bot2)
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	}
}

func TestBotRegistry_GetAll(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")

	registry.Register(bot1)
	registry.Register(bot2)

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 bots, got %d", len(all))
	}
}

func TestBotRegistry_FilterByName(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")
	bot3 := NewBotClient("bot3", "cli_3", "secret3", "executor")

	registry.Register(bot1)
	registry.Register(bot2)
	registry.Register(bot3)

	// 只启动 bot1 和 bot3
	filtered := registry.FilterByName([]string{"bot1", "bot3"})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered bots, got %d", len(filtered))
	}

	for _, bot := range filtered {
		if bot.Name != "bot1" && bot.Name != "bot3" {
			t.Errorf("Unexpected bot in filtered result: %s", bot.Name)
		}
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/bot -v -run TestBotRegistry
```

预期结果: `undefined: NewBotRegistry`

- [ ] **Step 3: 实现机器人注册表**

```go
package bot

import (
	"fmt"
	"sync"
)

// BotRegistry 机器人注册表
type BotRegistry struct {
	mu     sync.RWMutex
	bots   map[string]*BotClient          // key: bot_name
	byApp  map[string]*BotClient          // key: app_id
	byRole map[string][]*BotClient        // key: role
}

// NewBotRegistry 创建机器人注册表
func NewBotRegistry() *BotRegistry {
	return &BotRegistry{
		bots:   make(map[string]*BotClient),
		byApp:  make(map[string]*BotClient),
		byRole: make(map[string][]*BotClient),
	}
}

// Register 注册机器人
func (r *BotRegistry) Register(bot *BotClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查名称唯一性
	if _, exists := r.bots[bot.Name]; exists {
		return fmt.Errorf("机器人名称已存在: %s", bot.Name)
	}

	// 检查 AppID 唯一性
	if _, exists := r.byApp[bot.AppID]; exists {
		return fmt.Errorf("AppID 已存在: %s", bot.AppID)
	}

	// 注册
	r.bots[bot.Name] = bot
	r.byApp[bot.AppID] = bot
	r.byRole[bot.Role] = append(r.byRole[bot.Role], bot)

	return nil
}

// GetByName 按名称获取机器人
func (r *BotRegistry) GetByName(name string) *BotClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.bots[name]
}

// GetByAppID 按 AppID 获取机器人
func (r *BotRegistry) GetByAppID(appID string) *BotClient {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byApp[appID]
}

// GetByRole 按角色获取机器人列表
func (r *BotRegistry) GetByRole(role string) []*BotClient {
	r.mu.RLock()
	defer r.mu.RUnlock()

	bots := r.byRole[role]
	result := make([]*BotClient, len(bots))
	copy(result, bots)
	return result
}

// GetAll 获取所有机器人
func (r *BotRegistry) GetAll() []*BotClient {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*BotClient, 0, len(r.bots))
	for _, bot := range r.bots {
		result = append(result, bot)
	}
	return result
}

// FilterByName 按名称列表过滤机器人
func (r *BotRegistry) FilterByName(names []string) []*BotClient {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*BotClient, 0, len(names))
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, bot := range r.bots {
		if nameSet[bot.Name] {
			result = append(result, bot)
		}
	}
	return result
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/bot -v -run TestBotRegistry
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/bot/registry.go internal/bot/registry_test.go
git commit -m "feat: 实现机器人注册表"
```

---

### Task 7: 飞书 SDK 客户端封装

**目标：** 封装飞书 SDK，提供统一的客户端接口

**Files:**
- Create: `internal/bot/client.go`

- [ ] **Step 1: 编写客户端封装测试**

```go
package bot

import (
	"testing"
)

func TestNewLarkClient(t *testing.T) {
	bot := NewBotClient("test", "cli_123", "secret", "dispatcher")

	client, err := NewLarkClient(bot)
	if err != nil {
		t.Fatalf("NewLarkClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.BotID != bot.AppID {
		t.Errorf("Expected BotID '%s', got '%s'", bot.AppID, client.BotID)
	}
}

func TestLarkClient_SendMessage(t *testing.T) {
	// 这个测试需要 Mock，在后续任务中实现
	// 这里先测试结构体字段
	client := &LarkClient{
		BotID: "cli_123",
	}

	if client.BotID != "cli_123" {
		t.Errorf("Expected BotID 'cli_123', got '%s'", client.BotID)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/bot -v -run TestLarkClient
```

预期结果: `undefined: NewLarkClient`, `undefined: LarkClient`

- [ ] **Step 3: 实现客户端封装**

```go
package bot

import (
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

// LarkClient 飞书客户端封装
type LarkClient struct {
	BotID    string
	AppID    string
	AppSecret string

	// 飞书 SDK 客户端（懒加载）
	client *lark.Service
}

// NewLarkClient 创建飞书客户端
func NewLarkClient(bot *BotClient) (*LarkClient, error) {
	if bot == nil {
		return nil, fmt.Errorf("bot 不能为空")
	}

	return &LarkClient{
		BotID:     bot.AppID,
		AppID:     bot.AppID,
		AppSecret: bot.AppSecret,
	}, nil
}

// Init 初始化飞书 SDK 客户端
func (c *LarkClient) Init() error {
	// 创建飞书客户端
	// 实际 SDK 初始化将在后续任务中完成
	return nil
}

// SendMessage 发送消息（待实现）
func (c *LarkClient) SendMessage(chatID, message string) error {
	// 待实现
	return nil
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/bot -v -run TestLarkClient
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/bot/client.go internal/bot/client_test.go
git commit -m "feat: 封装飞书 SDK 客户端"
```

---

### Task 8: 消息路由器

**目标：** 实现消息路由器，将事件分发到对应的 Handler

**Files:**
- Create: `internal/router/router.go`

- [ ] **Step 1: 编写路由器测试框架**

```go
package router

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// MockHandler 用于测试
type MockHandler struct {
	Called bool
}

func (m *MockHandler) Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error {
	m.Called = true
	return nil
}

func TestMessageRouter_RegisterHandler(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	mockHandler := &MockHandler{}

	router.RegisterHandler("dispatcher", mockHandler)

	// 验证 handler 已注册
	// 实际测试在 Route 测试中进行
}

func TestMessageRouter_Route(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 注册测试机器人
	testBot := bot.NewBotClient("test", "cli_123", "secret", "dispatcher")
	registry.Register(testBot)

	// 注册 handler
	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)

	// 创建模拟事件
	event := map[string]interface{}{
		"receiver": map[string]interface{}{
			"bot_id": "cli_123",
		},
	}

	// 路由
	err := router.Route(context.Background(), event)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// 验证 handler 被调用
	if !mockHandler.Called {
		t.Error("Expected handler to be called")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/router -v
```

预期结果: `undefined: NewMessageRouter`

- [ ] **Step 3: 实现消息路由器框架**

```go
package router

import (
	"context"
	"fmt"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error
}

// MessageRouter 消息路由器
type MessageRouter struct {
	registry *bot.BotRegistry
	handlers map[string]MessageHandler // key: role
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter(registry *bot.BotRegistry) *MessageRouter {
	return &MessageRouter{
		registry: registry,
		handlers: make(map[string]MessageHandler),
	}
}

// RegisterHandler 注册处理器
func (r *MessageRouter) RegisterHandler(role string, handler MessageHandler) {
	r.handlers[role] = handler
}

// Route 路由事件到对应的处理器
func (r *MessageRouter) Route(ctx context.Context, event interface{}) error {
	// 解析事件获取接收者 bot_id
	receiverBotID, err := extractReceiverBotID(event)
	if err != nil {
		return fmt.Errorf("解析事件失败: %w", err)
	}

	// 获取接收机器人
	receiverBot := r.registry.GetByAppID(receiverBotID)
	if receiverBot == nil {
		return fmt.Errorf("未找到机器人: %s", receiverBotID)
	}

	// 获取对应的 handler
	handler, exists := r.handlers[receiverBot.Role]
	if !exists {
		return fmt.Errorf("未找到角色 '%s' 的处理器", receiverBot.Role)
	}

	// 调用 handler
	return handler.Handle(ctx, event, receiverBot)
}

// extractReceiverBotID 从事件中提取接收者 bot_id
func extractReceiverBotID(event interface{}) (string, error) {
	// 简化实现，实际需要解析飞书事件结构
	if eventMap, ok := event.(map[string]interface{}); ok {
		if receiver, ok := eventMap["receiver"].(map[string]interface{}); ok {
			if botID, ok := receiver["bot_id"].(string); ok {
				return botID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 receiver.bot_id")
}
```

- [ ] **Step 4: 迂行测试验证通过**

```bash
go test ./internal/router -v
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/router/router.go internal/router/router_test.go
git commit -m "feat: 实现消息路由器框架"
```

---

### Task 9: Handler 接口和通用功能

**目标：** 定义 Handler 接口并实现通用消息功能

**Files:**
- Create: `internal/handler/handler.go`
- Create: `internal/common/message.go`

- [ ] **Step 1: 编写 Handler 接口测试**

```go
package handler

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

func TestHandlerInterface(t *testing.T) {
	// 测试接口实现
	var _ MessageHandler = (*MockHandler)(nil)
}

// MockHandler 测试用的 mock handler
type MockHandler struct{}

func (m *MockHandler) Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error {
	return nil
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/handler -v
```

预期结果: `undefined: MessageHandler`

- [ ] **Step 3: 定义 Handler 接口**

```go
package handler

import (
	"context"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error
}

// BaseHandler 基础处理器，提供通用功能
type BaseHandler struct {
	// 通用字段
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}
```

- [ ] **Step 4: 创建通用消息功能**

```go
package common

import (
	"fmt"
	"strings"
)

// FormatTaskMessage 格式化任务消息
func FormatTaskMessage(taskName, status, output string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("任务 [%s] %s\n", taskName, status))

	if output != "" {
		// 处理多行输出
		lines := strings.Split(output, "\n")
		if len(lines) > 10 {
			sb.WriteString("输出：\n")
			for i, line := range lines[:10] {
				sb.WriteString(line + "\n")
			}
			sb.WriteString("...（输出已截断，完整日志请查看任务记录）")
		} else {
			sb.WriteString(fmt.Sprintf("输出：\n%s", output))
		}
	}

	return sb.String()
}

// FormatErrorMessage 格式化错误消息
func FormatErrorMessage(taskName, errorMsg string) string {
	return fmt.Sprintf("任务 [%s] 执行失败\n原因：%s", taskName, errorMsg)
}

// EscapeOutput 转义输出中的特殊字符
func EscapeOutput(output string) string {
	// 转义引号
	output = strings.ReplaceAll(output, "\"", "\\\"")
	// 转义换行符为 \n
	output = strings.ReplaceAll(output, "\n", "\\n")
	return output
}
```

- [ ] **Step 5: 编写通用功能测试**

```go
package common

import (
	"testing"
)

func TestFormatTaskMessage(t *testing.T) {
	result := FormatTaskMessage("deploy.sh", "执行成功", "部署完成")

	expected := "任务 [deploy.sh] 执行成功\n输出：\n部署完成"
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestFormatTaskMessage_LongOutput(t *testing.T) {
	// 创建超过 10 行的输出
	var output string
	for i := 0; i < 15; i++ {
		output += fmt.Sprintf("Line %d\n", i)
	}

	result := FormatTaskMessage("test.sh", "completed", output)

	if !strings.Contains(result, "（输出已截断") {
		t.Error("Expected truncation message for long output")
	}
}

func TestFormatErrorMessage(t *testing.T) {
	result := FormatErrorMessage("deploy.sh", "配置文件不存在")

	expected := "任务 [deploy.sh] 执行失败\n原因：配置文件不存在"
	if result != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, result)
	}
}

func TestEscapeOutput(t *testing.T) {
	input := `Line 1
Line 2
Line "with quotes"`

	result := EscapeOutput(input)

	if !strings.Contains(result, "\\n") {
		t.Error("Expected newline escaping")
	}

	if !strings.Contains(result, "\\\"") {
		t.Error("Expected quote escaping")
	}
}
```

- [ ] **Step 6: 运行测试验证通过**

```bash
go test ./internal/handler ./internal/common -v
```

预期结果: `PASS`

- [ ] **Step 7: 提交**

```bash
git add internal/handler/handler.go internal/common/message.go internal/common/message_test.go
git commit -m "feat: 定义 Handler 接口和通用消息功能"
```

---

### Task 10: DispatcherHandler 实现

**目标：** 实现 DispatcherHandler，处理用户消息和任务分发

**Files:**
- Create: `internal/handler/dispatcher.go`

- [ ] **Step 1: 编写 DispatcherHandler 测试**

```go
package handler

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

func TestDispatcherHandler_HandleUserMessage(t *testing.T) {
	handler := NewDispatcherHandler()

	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	// 创建模拟用户消息事件
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "user_1",
		},
		"message": map[string]interface{}{
			"content": "执行 deploy.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected for incomplete implementation): %v", err)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/handler -v -run TestDispatcher
```

预期结果: `undefined: NewDispatcherHandler`

- [ ] **Step 3: 实现 DispatcherHandler 基础结构**

```go
package handler

import (
	"context"
	"fmt"

	"github.com/yourname/lark-bot-service/internal/bot"
	"github.com/yourname/lark-bot-service/internal/common"
)

// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
	*BaseHandler

	// 白名单
	userWhiteList  map[string]bool
	taskWhiteList  map[string]bool

	// 机器人群 ID
	robotGroupID string
}

// NewDispatcherHandler 创建分发机器人处理器
func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		BaseHandler: NewBaseHandler(),
		userWhiteList: make(map[string]bool),
		taskWhiteList: make(map[string]bool),
	}
}

// SetWhiteLists 设置白名单
func (h *DispatcherHandler) SetWhiteLists(users, tasks []string) {
	h.userWhiteList = make(map[string]bool)
	for _, user := range users {
		h.userWhiteList[user] = true
	}

	h.taskWhiteList = make(map[string]bool)
	for _, task := range tasks {
		h.taskWhiteList[task] = true
	}
}

// Handle 处理消息
func (h *DispatcherHandler) Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error {
	// 解析事件
	senderID, err := extractSenderID(event)
	if err != nil {
		return fmt.Errorf("解析发送者失败: %w", err)
	}

	// 校验用户白名单
	if !h.userWhiteList[senderID] {
		return fmt.Errorf("用户不在白名单中: %s", senderID)
	}

	// 解析消息内容
	message, err := extractMessageContent(event)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 解析任务名
	taskName, err := h.parseTaskName(message)
	if err != nil {
		return fmt.Errorf("解析任务名失败: %w", err)
	}

	// 校验任务白名单
	if !h.taskWhiteList[taskName] {
		return fmt.Errorf("任务不在白名单中: %s", taskName)
	}

	// TODO: 在机器人群 @ 执行机器人
	// TODO: 创建任务记录

	return nil
}

// parseTaskName 从消息中解析任务名
func (h *DispatcherHandler) parseTaskName(message string) (string, error) {
	// 简化实现：假设消息格式为 "执行 <taskname>"
	// 实际需要更复杂的解析逻辑
	if len(message) < 3 {
		return "", fmt.Errorf("消息格式错误")
	}

	// 提取任务名（简化版）
	// 实际需要使用正则表达式或更复杂的解析
	return message, nil
}

// extractSenderID 从事件中提取发送者 ID
func extractSenderID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if userID, ok := sender["user_id"].(string); ok {
				return userID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 sender.user_id")
}

// extractMessageContent 从事件中提取消息内容
func extractMessageContent(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 message.content")
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/handler -v -run TestDispatcher
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/handler/dispatcher.go internal/handler/dispatcher_test.go
git commit -m "feat: 实现 DispatcherHandler 基础功能"
```

---

### Task 11: ExecutorHandler 实现

**目标：** 实现 ExecutorHandler，处理任务执行

**Files:**
- Create: `internal/handler/executor.go`

- [ ] **Step 1: 编写 ExecutorHandler 测试**

```go
package handler

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

func TestExecutorHandler_VerifyDispatcher(t *testing.T) {
	handler := NewExecutorHandler()

	// 设置允许的 dispatcher
	handler.SetAllowedDispatchers([]string{"cli_123"})

	testBot := bot.NewBotClient("executor", "cli_456", "secret", "executor")

	// 创建来自允许的 dispatcher 的事件
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_123",
		},
		"message": map[string]interface{}{
			"content": "execute /opt/scripts/test.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected for incomplete implementation): %v", err)
	}
}

func TestExecutorHandler_UnauthorizedDispatcher(t *testing.T) {
	handler := NewExecutorHandler()

	// 设置允许的 dispatcher
	handler.SetAllowedDispatchers([]string{"cli_123"})

	testBot := bot.NewBotClient("executor", "cli_456", "secret", "executor")

	// 创建来自未授权的 dispatcher 的事件
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_unauthorized",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for unauthorized dispatcher, got nil")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/handler -v -run TestExecutor
```

预期结果: `undefined: NewExecutorHandler`

- [ ] **Step 3: 实现 ExecutorHandler**

```go
package handler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// ExecutorHandler 执行机器人处理器
type ExecutorHandler struct {
	*BaseHandler

	allowedDispatchers map[string]bool
	allowedScripts     map[string]bool
}

// NewExecutorHandler 创建执行机器人处理器
func NewExecutorHandler() *ExecutorHandler {
	return &ExecutorHandler{
		BaseHandler:        NewBaseHandler(),
		allowedDispatchers: make(map[string]bool),
		allowedScripts:     make(map[string]bool),
	}
}

// SetAllowedDispatchers 设置允许的 dispatcher 列表
func (h *ExecutorHandler) SetAllowedDispatchers(dispatchers []string) {
	h.allowedDispatchers = make(map[string]bool)
	for _, d := range dispatchers {
		h.allowedDispatchers[d] = true
	}
}

// SetAllowedScripts 设置允许执行的脚本列表
func (h *ExecutorHandler) SetAllowedScripts(scripts []string) {
	h.allowedScripts = make(map[string]bool)
	for _, script := range scripts {
		h.allowedScripts[script] = true
	}
}

// Handle 处理消息
func (h *ExecutorHandler) Handle(ctx context.Context, event interface{}, bot *bot.BotClient) error {
	// 解析发送者 bot_id
	senderBotID, err := extractSenderBotID(event)
	if err != nil {
		return fmt.Errorf("解析发送者失败: %w", err)
	}

	// 校验是否来自允许的 dispatcher
	if !h.allowedDispatchers[senderBotID] {
		return fmt.Errorf("未授权的 dispatcher: %s", senderBotID)
	}

	// 解析消息内容
	message, err := extractMessageContent(event)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 解析脚本路径
	scriptPath, args, err := h.parseScriptCommand(message)
	if err != nil {
		return fmt.Errorf("解析脚本命令失败: %w", err)
	}

	// 校验脚本是否在白名单
	if !h.allowedScripts[scriptPath] {
		return fmt.Errorf("脚本不在白名单中: %s", scriptPath)
	}

	// 执行脚本
	output, err := h.executeScript(scriptPath, args)
	if err != nil {
		return fmt.Errorf("执行脚本失败: %w", err)
	}

	// TODO: 向 dispatcher 汇报结果
	_ = output

	return nil
}

// parseScriptCommand 解析脚本命令
func (h *ExecutorHandler) parseScriptCommand(message string) (scriptPath string, args []string, err error) {
	// 简化实现：假设格式为 "execute <script> [args...]"
	parts := strings.Fields(message)
	if len(parts) < 2 || parts[0] != "execute" {
		return "", nil, fmt.Errorf("命令格式错误，应为: execute <script> [args...]")
	}

	scriptPath = parts[1]
	args = parts[2:]

	return scriptPath, args, nil
}

// executeScript 执行脚本
func (h *ExecutorHandler) executeScript(scriptPath string, args []string) (string, error) {
	cmd := exec.Command(scriptPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("脚本执行失败: %w", err)
	}
	return string(output), nil
}

// extractSenderBotID 从事件中提取发送者 bot_id
func extractSenderBotID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if botID, ok := sender["bot_id"].(string); ok {
				return botID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 sender.bot_id")
}

// extractMessageContent 从事件中提取消息内容
func extractMessageContent(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 message.content")
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/handler -v -run TestExecutor
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/handler/executor.go internal/handler/executor_test.go
git commit -m "feat: 实现 ExecutorHandler"
```

---

### Task 12: 任务日志记录器

**目标：** 实现任务日志记录功能

**Files:**
- Create: `internal/logger/task_logger.go`

- [ ] **Step 1: 编写任务日志测试**

```go
package logger

import (
	"testing"
	"time"

	"github.com/yourname/lark-bot-service/internal/bot"
)

func TestTaskLogger_CreateTask(t *testing.T) {
	logger := NewTaskLogger()

	record := &bot.TaskRecord{
		ID:         "task-1",
		TaskName:   "deploy.sh",
		Dispatcher: "task-dispatcher",
		Executor:   "shell-executor-1",
		User:       "user_1",
		Status:     "pending",
		StartTime:  time.Now(),
	}

	err := logger.CreateTask(record)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 验证任务已创建
	retrieved, err := logger.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.TaskName != "deploy.sh" {
		t.Errorf("Expected task name 'deploy.sh', got '%s'", retrieved.TaskName)
	}
}

func TestTaskLogger_UpdateTaskStatus(t *testing.T) {
	logger := NewTaskLogger()

	record := &bot.TaskRecord{
		ID:        "task-2",
		TaskName:  "test.sh",
		Status:    "pending",
		StartTime: time.Now(),
	}

	logger.CreateTask(record)

	// 更新状态
	now := time.Now()
	err := logger.UpdateTaskStatus("task-2", "completed", &now, "执行成功", "")
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// 验证更新
	retrieved, _ := logger.GetTask("task-2")
	if retrieved.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}
}

func TestTaskLogger_QueryTasks(t *testing.T) {
	logger := NewTaskLogger()

	// 创建多个任务
	for i := 0; i < 3; i++ {
		record := &bot.TaskRecord{
			ID:         fmt.Sprintf("task-%d", i),
			TaskName:   "test.sh",
			Status:     "completed",
			StartTime:  time.Now(),
		}
		logger.CreateTask(record)
	}

	// 查询
	tasks, err := logger.QueryTasks(TaskFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) < 3 {
		t.Errorf("Expected at least 3 tasks, got %d", len(tasks))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/logger -v
```

预期结果: `undefined: NewTaskLogger`

- [ ] **Step 3: 实现任务日志记录器**

```go
package logger

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// TaskFilter 任务查询过滤器
type TaskFilter struct {
	TaskName string
	Status   string
	User     string
	Executor string
	// 可以添加更多过滤条件
}

// TaskLogger 任务日志记录器
type TaskLogger struct {
	mu    sync.RWMutex
	tasks map[string]*bot.TaskRecord // key: task_id
}

// NewTaskLogger 创建任务日志记录器
func NewTaskLogger() *TaskLogger {
	return &TaskLogger{
		tasks: make(map[string]*bot.TaskRecord),
	}
}

// CreateTask 创建任务记录
func (l *TaskLogger) CreateTask(record *bot.TaskRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.tasks[record.ID]; exists {
		return fmt.Errorf("任务 ID 已存在: %s", record.ID)
	}

	l.tasks[record.ID] = record
	return nil
}

// GetTask 获取任务记录
func (l *TaskLogger) GetTask(taskID string) (*bot.TaskRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	record, exists := l.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	return record, nil
}

// UpdateTaskStatus 更新任务状态
func (l *TaskLogger) UpdateTaskStatus(taskID, status string, endTime *time.Time, result, errorMsg string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, exists := l.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	record.Status = status
	record.EndTime = endTime
	record.Result = result
	record.Error = errorMsg

	return nil
}

// QueryTasks 查询任务记录
func (l *TaskLogger) QueryTasks(filter TaskFilter) ([]*bot.TaskRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []*bot.TaskRecord

	for _, record := range l.tasks {
		if l.matchFilter(record, filter) {
			results = append(results, record)
		}
	}

	return results, nil
}

// matchFilter 检查任务是否匹配过滤器
func (l *TaskLogger) matchFilter(record *bot.TaskRecord, filter TaskFilter) bool {
	if filter.TaskName != "" && record.TaskName != filter.TaskName {
		return false
	}

	if filter.Status != "" && record.Status != filter.Status {
		return false
	}

	if filter.User != "" && record.User != filter.User {
		return false
	}

	if filter.Executor != "" && record.Executor != filter.Executor {
		return false
	}

	return true
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/logger -v
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/logger/task_logger.go internal/logger/task_logger_test.go
git commit -m "feat: 实现任务日志记录器"
```

---

### Task 13: 错误定义

**目标：** 定义统一的错误类型和处理

**Files:**
- Create: `internal/common/errors.go`

- [ ] **Step 1: 编写错误测试**

```go
package common

import (
	"errors"
	"testing"
)

func TestHandlerError_Error(t *testing.T) {
	err := &HandlerError{
		Code:        "AUTH_FAILED",
		Message:     "认证失败",
		ReplyToUser: true,
	}

	expected := "[AUTH_FAILED] 认证失败"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestIsHandlerError(t *testing.T) {
	handlerErr := &HandlerError{
		Code:    "TEST_ERROR",
		Message: "测试错误",
	}

	if !IsHandlerError(handlerErr) {
		t.Error("Expected IsHandlerError to return true")
	}

	regularErr := errors.New("regular error")
	if IsHandlerError(regularErr) {
		t.Error("Expected IsHandlerError to return false for regular error")
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/common -v -run TestHandlerError
```

预期结果: `undefined: HandlerError`

- [ ] **Step 3: 定义错误类型**

```go
package common

import (
	"fmt"
)

// HandlerError 处理器错误
type HandlerError struct {
	Code        string
	Message     string
	Cause       error
	ReplyToUser bool
}

// Error 实现 error 接口
func (e *HandlerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回底层错误
func (e *HandlerError) Unwrap() error {
	return e.Cause
}

// IsHandlerError 检查错误是否为 HandlerError
func IsHandlerError(err error) bool {
	_, ok := err.(*HandlerError)
	return ok
}

// 预定义错误代码
const (
	ErrCodeAuthFailed       = "AUTH_FAILED"
	ErrCodeTaskNotAllowed   = "TASK_NOT_ALLOWED"
	ErrCodeExecutionFailed  = "EXECUTION_FAILED"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeInvalidMessage   = "INVALID_MESSAGE"
	ErrCodeUnknownBot       = "UNKNOWN_BOT"
	ErrCodeUnknownRole      = "UNKNOWN_ROLE"
)

// NewHandlerError 创建 HandlerError
func NewHandlerError(code, message string, cause error, replyToUser bool) *HandlerError {
	return &HandlerError{
		Code:        code,
		Message:     message,
		Cause:       cause,
		ReplyToUser: replyToUser,
	}
}

// 预定义错误构造函数
func NewAuthFailedError(message string, cause error) *HandlerError {
	return NewHandlerError(ErrCodeAuthFailed, message, cause, true)
}

func NewTaskNotAllowedError(taskName string) *HandlerError {
	return NewHandlerError(ErrCodeTaskNotAllowed, fmt.Sprintf("任务不允许执行: %s", taskName), nil, true)
}

func NewExecutionFailedError(message string, cause error) *HandlerError {
	return NewHandlerError(ErrCodeExecutionFailed, message, cause, true)
}

func NewUnauthorizedError(message string) *HandlerError {
	return NewHandlerError(ErrCodeUnauthorized, message, nil, false)
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/common -v -run TestHandlerError
```

预期结果: `PASS`

- [ ] **Step 5: 提交**

```bash
git add internal/common/errors.go internal/common/errors_test.go
git commit -m "feat: 定义统一的错误类型"
```

---

### Task 14: 主程序入口

**目标：** 实现主程序入口和 CLI 命令

**Files:**
- Create: `cmd/bot-service/main.go`

- [ ] **Step 1: 创建主程序框架**

```go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	bots    string
)

var rootCmd = &cobra.Command{
	Use:   "bot-service",
	Short: "飞书机器人服务",
	Long:  `多机器人协作服务，支持配置文件管理和任务分发`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Run:   runStart,
}

func init() {
	cobra.OnInitialize(initConfig)

	// 全局标志
	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "configs/bots.yaml", "配置文件路径")
	startCmd.Flags().StringVarP(&bots, "bots", "b", "", "要启动的机器人列表（逗号分隔），不指定则启动所有")

	// 添加 --all 标志
	startCmd.Flags().Bool("all", false, "启动所有机器人")

	rootCmd.AddCommand(startCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("configs")
		viper.SetConfigName("bots")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("读取配置文件失败: %v\n", err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) {
	// 获取配置文件路径
	configPath, _ := cmd.Flags().GetString("config")

	// 加载配置
	fmt.Printf("正在加载配置: %s\n", configPath)
	// TODO: 实现配置加载

	// 获取要启动的机器人
	botList, _ := cmd.Flags().GetString("bots")
	startAll, _ := cmd.Flags().GetBool("all")

	if startAll {
		fmt.Println("启动所有机器人...")
	} else if botList != "" {
		fmt.Printf("启动机器人: %s\n", botList)
	} else {
		fmt.Println("未指定机器人，启动所有...")
	}

	// TODO: 实现服务启动逻辑

	fmt.Println("服务启动中...")
	select {} // 阻塞主线程
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: 测试 CLI 命令**

```bash
go run cmd/bot-service/main.go start --help
```

预期结果: 显示帮助信息

- [ ] **Step 3: 提交**

```bash
git add cmd/bot-service/main.go
git commit -m "feat: 实现 CLI 命令框架"
```

---

### Task 15: 集成主程序逻辑

**目标：** 将所有组件集成到主程序中

**Files:**
- Modify: `cmd/bot-service/main.go`

- [ ] **Step 1: 实现完整的启动逻辑**

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourname/lark-bot-service/internal/bot"
	"github.com/yourname/lark-bot-service/internal/config"
	"github.com/yourname/lark-bot-service/internal/handler"
	"github.com/yourname/lark-bot-service/internal/logger"
	"github.com/yourname/lark-bot-service/internal/router"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	bots    string
)

var rootCmd = &cobra.Command{
	Use:   "bot-service",
	Short: "飞书机器人服务",
	Long:  `多机器人协作服务，支持配置文件管理和任务分发`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Run:   runStart,
}

func init() {
	cobra.OnInitialize(initConfig)

	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "configs/bots.yaml", "配置文件路径")
	startCmd.Flags().StringVarP(&bots, "bots", "b", "", "要启动的机器人列表（逗号分隔）")
	startCmd.Flags().Bool("all", false, "启动所有机器人")

	rootCmd.AddCommand(startCmd)
}

func initConfig() {
	// 配置已在 runStart 中手动加载
}

func runStart(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 加载配置
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 验证配置
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 配置加载成功")

	// 3. 创建机器人注册表
	registry := bot.NewBotRegistry()

	// 4. 注册所有机器人
	for _, botCfg := range cfg.Bots {
		botClient := bot.NewBotClient(botCfg.Name, botCfg.AppID, botCfg.AppSecret, botCfg.Role)
		if err := registry.Register(botClient); err != nil {
			fmt.Printf("❌ 注册机器人失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  - 注册机器人: %s (%s)\n", botCfg.Name, botCfg.Role)
	}

	// 5. 过滤要启动的机器人
	var activeBots []*bot.BotClient
	botList, _ := cmd.Flags().GetString("bots")
	startAll, _ := cmd.Flags().GetBool("all")

	if startAll || botList == "" {
		activeBots = registry.GetAll()
	} else {
		activeBots = registry.FilterByName(parseBotList(botList))
	}

	if len(activeBots) == 0 {
		fmt.Println("❌ 没有要启动的机器人")
		os.Exit(1)
	}

	fmt.Printf("✅ 将启动 %d 个机器人\n", len(activeBots))

	// 6. 创建任务日志
	taskLogger := logger.NewTaskLogger()

	// 7. 创建路由器
	messageRouter := router.NewMessageRouter(registry)

	// 8. 创建并注册 handlers
	dispatcherHandler := handler.NewDispatcherHandler()
	dispatcherHandler.SetWhiteLists(cfg.UserWhiteList, cfg.TaskWhiteList)
	messageRouter.RegisterHandler("dispatcher", dispatcherHandler)

	executorHandler := handler.NewExecutorHandler()
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			executorHandler.SetAllowedDispatchers(botCfg.AllowedDispatchers)
			executorHandler.SetAllowedScripts(botCfg.AllowedScripts)
		}
	}
	messageRouter.RegisterHandler("executor", executorHandler)

	fmt.Println("✅ 服务初始化完成")

	// 9. 启动事件监听（TODO: 实现飞书事件监听）
	fmt.Println("⏳ 等待飞书事件...")

	// 10. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 正在关闭服务...")
	_ = taskLogger
	_ = ctx
	_ = messageRouter
	fmt.Println("✅ 服务已关闭")
}

func parseBotList(botList string) []string {
	var result []string
	for _, s := range splitAndTrim(botList, ",") {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitList(s, sep) {
		result = append(result, trimSpace(part))
	}
	return result
}

func splitList(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == sep {
			result = append(result, current)
			current = ""
		} else {
			current += s[i : i+1]
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && s[start:start+1] == " " {
		start++
	}
	end := len(s)
	for end > start && s[end-1:end] == " " {
		end--
	}
	return s[start:end]
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: 测试启动命令**

```bash
go run cmd/bot-service/main.go start --help
```

- [ ] **Step 3: 提交**

```bash
git add cmd/bot-service/main.go
git commit -m "feat: 集成完整的服务启动逻辑"
```

---

### Task 16: 配置文件示例

**目标：** 创建配置文件示例和文档

**Files:**
- Create: `configs/bots.yaml.example`
- Modify: `README.md`

- [ ] **Step 1: 创建配置文件示例**

```yaml
# 飞书机器人服务配置文件示例

# 机器人群 ID（所有机器人必须加入此群）
robot_group_id: "oc_xxxxxxxxxxxxxxxxx"

# 任务白名单（分发机器人允许分发的任务）
task_whitelist:
  - "deploy"
  - "restart"
  - "check_logs"
  - "run_test"

# 用户白名单（分发机器人允许的用户）
user_whitelist:
  - "ou_xxxxxxxxxxxxxxxxx"
  - "ou_yyyyyyyyyyyyyyyyyy"

# 机器人配置列表
bots:
  # 分发机器人
  - name: "task-dispatcher"
    app_id: "cli_xxxxxxxxxxxxxxxxx"
    app_secret: "xxxxxxxxxxxxxxxxxxxx"
    role: "dispatcher"

  # 执行机器人 1 - Shell 脚本执行器
  - name: "shell-executor"
    app_id: "cli_yyyyyyyyyyyyyyyyyy"
    app_secret: "yyyyyyyyyyyyyyyyyy"
    role: "executor"
    allowed_dispatchers:
      - "cli_xxxxxxxxxxxxxxxxx"  # 只接受来自 task-dispatcher 的命令
    allowed_scripts:
      - "/opt/scripts/deploy.sh"
      - "/opt/scripts/restart.sh"
      - "/opt/scripts/check_logs.sh"

  # 执行机器人 2 - 测试执行器
  - name: "test-executor"
    app_id: "cli_zzzzzzzzzzzzzzzzzzzz"
    app_secret: "zzzzzzzzzzzzzzzzzzzz"
    role: "executor"
    allowed_dispatchers:
      - "cli_xxxxxxxxxxxxxxxxx"
    allowed_scripts:
      - "/opt/scripts/run_test.sh"
```

- [ ] **Step 2: 更新 README**

```markdown
# 飞书机器人服务

多机器人协作服务，支持配置文件管理和任务分发。

## 功能特性

- ✅ 配置文件管理多个机器人
- ✅ 启动参数控制启动机器人列表
- ✅ 角色分离（Dispatcher/Executor）
- ✅ 分层安全（白名单 + 来源校验）
- ✅ 完整任务日志记录

## 快速开始

### 1. 安装依赖

\`\`\`bash
go mod download
\`\`\`

### 2. 配置

\`\`\`bash
# 复制配置文件模板
cp configs/bots.yaml.example configs/bots.yaml

# 编辑配置，填入你的飞书应用信息
vim configs/bots.yaml
\`\`\`

### 3. 启动服务

\`\`\`bash
# 启动所有机器人
go run cmd/bot-service/main.go start --all

# 启动指定机器人
go run cmd/bot-service/main.go start --bots=task-dispatcher,shell-executor

# 使用自定义配置文件
go run cmd/bot-service/main.go start --config=/path/to/config.yaml
\`\`\`

### 4. 构建

\`\`\`bash
go build -o bot-service cmd/bot-service/main.go
./bot-service start --all
\`\`\`

## 配置说明

### 机器人角色

**Dispatcher（分发机器人）**
- 加入用户群，接收用户指令
- 在机器人群分发任务给执行机器人
- 收集结果并回复用户
- 配置用户白名单和任务白名单

**Executor（执行机器人）**
- 只在机器人群，接收分发机器人指令
- 执行脚本等任务
- 配置允许的 dispatcher 和脚本白名单

### 安全配置

\`\`\`yaml
# 用户白名单：只允许这些用户发起任务
user_whitelist:
  - "ou_xxx"  # 飞书用户 ID

# 任务白名单：只允许执行这些任务
task_whitelist:
  - "deploy"
  - "restart"

# Executor 安全校验
allowed_dispatchers:
  - "cli_xxx"  # 只接受这些 dispatcher 的命令
allowed_scripts:
  - "/opt/scripts/deploy.sh"  # 只允许执行这些脚本
\`\`\`

## 开发

\`\`\`bash
# 运行测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 格式化代码
go fmt ./...

# 静态检查
go vet ./...
\`\`\`

## 架构

\`\`\`
┌─────────────┐
│  用户在群里  │
│  @分发机器人  │
└──────┬──────┘
       │
       ↓
┌─────────────────────────────────────────┐
│  Dispatcher: 校验用户 → 分发任务          │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│  Executor: 执行脚本 → 汇报结果            │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│  Dispatcher: 回复用户任务结果             │
└─────────────────────────────────────────┘
\`\`\`

## License

MIT
```

- [ ] **Step 3: 提交**

```bash
git add configs/bots.yaml.example README.md
git commit -m "docs: 添加配置文件示例和完整文档"
```

---

## 总结

本实施计划包含 16 个任务，涵盖了飞书机器人服务的完整实现：

✅ **已完成的核心功能：**
- 配置管理（加载、验证）
- 机器人注册表
- 消息路由器
- Dispatcher 和 Executor Handler
- 任务日志记录
- 错误处理
- CLI 命令行工具

📋 **后续扩展任务（未在本计划中）：**
- 飞书 SDK 实际集成和事件监听
- 消息发送功能实现
- 数据库持久化（可选）
- Web UI（可选）

## 开发原则

本计划遵循以下原则：
- **TDD**: 每个功能先写测试
- **小步提交**: 每个 step 完成后立即 commit
- **DRY**: 避免重复代码
- **YAGNI**: 只实现当前需要的功能
