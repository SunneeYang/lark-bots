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
