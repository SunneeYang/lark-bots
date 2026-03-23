package config

import (
	"fmt"
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
