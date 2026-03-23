package bot

import (
	"fmt"
	"sync"
)

// BotRegistry 机器人注册表
type BotRegistry struct {
	mu     sync.RWMutex
	bots   map[string]*BotClient // key: bot_name
	byApp  map[string]*BotClient // key: app_id
	byRole map[string][]*BotClient // key: role
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

	if len(names) == 0 {
		return []*BotClient{}
	}

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
