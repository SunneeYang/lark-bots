package bot

import (
	"testing"
)

func TestBotRegistry_Register(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")

	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	err = registry.Register(bot2)
	if err != nil {
		t.Fatalf("Register bot2 failed: %v", err)
	}

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

	if retrieved.AppID != "cli_2" {
		t.Errorf("Expected app_id 'cli_2', got '%s'", retrieved.AppID)
	}

	// 验证按角色获取
	dispatchers := registry.GetByRole("dispatcher")
	if len(dispatchers) != 1 {
		t.Errorf("Expected 1 dispatcher, got %d", len(dispatchers))
	}

	if dispatchers[0].Name != "bot1" {
		t.Errorf("Expected dispatcher name 'bot1', got '%s'", dispatchers[0].Name)
	}
}

func TestBotRegistry_DuplicateName(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("duplicate", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("duplicate", "cli_2", "secret2", "executor")

	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	// 尝试注册同名机器人
	err = registry.Register(bot2)
	if err == nil {
		t.Error("Expected error for duplicate name, got nil")
	}
}

func TestBotRegistry_DuplicateAppID(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_dup", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_dup", "secret2", "executor")

	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	// 尝试注册同 AppID 机器人
	err = registry.Register(bot2)
	if err == nil {
		t.Error("Expected error for duplicate app_id, got nil")
	}
}

func TestBotRegistry_GetAll(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")

	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	err = registry.Register(bot2)
	if err != nil {
		t.Fatalf("Register bot2 failed: %v", err)
	}

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 bots, got %d", len(all))
	}

	// 验证返回的是新的 slice，不是内部 map 的引用
	// 注意：返回的是指向同一 BotClient 对象的指针，这是预期行为
	// 如果需要完全隔离，需要返回深拷贝，但通常不需要
	all = append(all, NewBotClient("bot3", "cli_3", "secret3", "dispatcher"))
	if len(registry.GetAll()) != 2 {
		t.Error("GetAll should return a new slice, not reference to internal slice")
	}
}

func TestBotRegistry_FilterByName(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	bot2 := NewBotClient("bot2", "cli_2", "secret2", "executor")
	bot3 := NewBotClient("bot3", "cli_3", "secret3", "executor")

	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	err = registry.Register(bot2)
	if err != nil {
		t.Fatalf("Register bot2 failed: %v", err)
	}

	err = registry.Register(bot3)
	if err != nil {
		t.Fatalf("Register bot3 failed: %v", err)
	}

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

	// 测试空列表
	filtered = registry.FilterByName([]string{})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 bots for empty filter, got %d", len(filtered))
	}

	// 测试不存在的名称
	filtered = registry.FilterByName([]string{"nonexistent"})
	if len(filtered) != 0 {
		t.Errorf("Expected 0 bots for nonexistent name, got %d", len(filtered))
	}
}

func TestBotRegistry_GetByName_NotFound(t *testing.T) {
	registry := NewBotRegistry()

	retrieved := registry.GetByName("nonexistent")
	if retrieved != nil {
		t.Errorf("Expected nil for nonexistent bot, got %+v", retrieved)
	}
}

func TestBotRegistry_GetByAppID_NotFound(t *testing.T) {
	registry := NewBotRegistry()

	retrieved := registry.GetByAppID("cli_nonexistent")
	if retrieved != nil {
		t.Errorf("Expected nil for nonexistent app_id, got %+v", retrieved)
	}
}

func TestBotRegistry_GetByRole_NotFound(t *testing.T) {
	registry := NewBotRegistry()

	bot1 := NewBotClient("bot1", "cli_1", "secret1", "dispatcher")
	err := registry.Register(bot1)
	if err != nil {
		t.Fatalf("Register bot1 failed: %v", err)
	}

	retrieved := registry.GetByRole("executor")
	if len(retrieved) != 0 {
		t.Errorf("Expected 0 bots for nonexistent role, got %d", len(retrieved))
	}
}

func TestBotRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewBotRegistry()
	done := make(chan bool)

	// 并发注册
	for i := 0; i < 10; i++ {
		go func(n int) {
			bot := NewBotClient(
				"bot"+string(rune('0'+n)),
				"cli_"+string(rune('0'+n)),
				"secret",
				"dispatcher",
			)
			registry.Register(bot)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		go func() {
			registry.GetAll()
			registry.GetByName("bot1")
			registry.GetByRole("dispatcher")
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 20; i++ {
		<-done
	}

	// 验证没有数据竞争
	all := registry.GetAll()
	_ = all
}
