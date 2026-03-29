package config

import "fmt"

// ResolveUsers 将所有 TaskDetail.AllowedUsers 中的姓名解析为 open_id
// 必须在 ValidateConfig 之后调用
func ResolveUsers(cfg *ServiceConfig) error {
	for i := range cfg.Bots {
		for taskName, task := range cfg.Bots[i].Tasks {
			if len(task.AllowedUsers) == 0 {
				continue
			}
			resolved := make([]string, 0, len(task.AllowedUsers))
			for _, name := range task.AllowedUsers {
				if cfg.Users == nil {
					return fmt.Errorf("bot %q task %q: user %q not found in global users mapping (users map is empty)",
						cfg.Bots[i].Name, taskName, name)
				}
				openID, ok := cfg.Users[name]
				if !ok {
					return fmt.Errorf("bot %q task %q: user %q not found in global users mapping",
						cfg.Bots[i].Name, taskName, name)
				}
				if openID == "" {
					return fmt.Errorf("bot %q task %q: user %q has empty open_id in global users mapping",
						cfg.Bots[i].Name, taskName, name)
				}
				resolved = append(resolved, openID)
			}
			task.AllowedUsers = resolved
			cfg.Bots[i].Tasks[taskName] = task
		}
	}
	return nil
}
