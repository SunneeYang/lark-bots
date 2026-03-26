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

	// 检查所有 executor 的任务名唯一性（兼容新旧两种配置格式）
	// 旧格式: TaskScripts map[string]string → 任务名即为 key
	// 旧格式: LegacyTasks []ExecutorTask → 任务名取 Task.Name
	// 新格式: Tasks map[string]TaskDetail → 任务名即为 map key
	taskNames := make(map[string]string) // taskName -> executorName
	for _, bot := range cfg.Bots {
		if bot.Role == "executor" {
			// 旧格式 TaskScripts
			for taskName := range bot.TaskScripts {
				if existingExecutor, exists := taskNames[taskName]; exists {
					return fmt.Errorf("配置错误：任务名 '%s' 被多个 executor 声明（%s 和 %s），任务名必须全局唯一",
						taskName, existingExecutor, bot.Name)
				}
				taskNames[taskName] = bot.Name
			}
			// 旧格式 LegacyTasks (分层匹配)
			for _, task := range bot.LegacyTasks {
				if len(task.Names) == 0 {
					return fmt.Errorf("配置错误：executor '%s' 的任务缺少 names 配置", bot.Name)
				}
				if task.Name == "" {
					return fmt.Errorf("配置错误：executor '%s' 的任务缺少 name 配置", bot.Name)
				}
				taskName := task.Name // 任务名称
				if existingExecutor, exists := taskNames[taskName]; exists {
					return fmt.Errorf("配置错误：任务名 '%s' 被多个 executor 声明（%s 和 %s），任务名必须全局唯一",
						taskName, existingExecutor, bot.Name)
				}
				taskNames[taskName] = bot.Name
			}
			// 检查 LegacyTasks 中 keywords 不能为空
			for _, task := range bot.LegacyTasks {
				if len(task.Keywords) == 0 {
					return fmt.Errorf("配置错误：executor '%s' 的任务缺少 keywords 配置", bot.Name)
				}
				if task.Script == "" {
					return fmt.Errorf("配置错误：executor '%s' 的任务缺少 script 配置", bot.Name)
				}
			}
			// 新格式 Tasks (参数化任务)
			for taskName := range bot.Tasks {
				if existingExecutor, exists := taskNames[taskName]; exists {
					return fmt.Errorf("配置错误：任务名 '%s' 被多个 executor 声明（%s 和 %s），任务名必须全局唯一",
						taskName, existingExecutor, bot.Name)
				}
				taskNames[taskName] = bot.Name
				// 验证 TaskDetail 必填字段
				taskDetail := bot.Tasks[taskName]
				if taskDetail.Script == "" {
					return fmt.Errorf("配置错误：executor '%s' 的任务 '%s' 缺少 script 配置", bot.Name, taskName)
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
