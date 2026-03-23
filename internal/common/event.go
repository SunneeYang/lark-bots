package common

import (
	"encoding/json"
	"fmt"
)

// ExtractSenderID 从事件中提取发送者 ID（支持多种格式）
func ExtractSenderID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			// 格式 1: sender.user_id (旧格式)
			if userID, ok := sender["user_id"].(string); ok && userID != "" {
				return userID, nil
			}
			// 格式 2: sender.sender_id.open_id (飞书 SDK 格式)
			if senderID, ok := sender["sender_id"].(map[string]interface{}); ok {
				if openID, ok := senderID["open_id"].(string); ok && openID != "" {
					return openID, nil
				}
				if userID, ok := senderID["user_id"].(string); ok && userID != "" {
					return userID, nil
				}
			}
			// 格式 3: sender.sender_id (直接字符串)
			if senderID, ok := sender["sender_id"].(string); ok && senderID != "" {
				return senderID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取发送者 ID")
}

// ExtractSenderBotID 从事件中提取发送者 bot_id
func ExtractSenderBotID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if botID, ok := sender["bot_id"].(string); ok && botID != "" {
				return botID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 sender.bot_id")
}

// ExtractMessageContent 从事件中提取消息内容
func ExtractMessageContent(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 message.content")
}

// ParseMessageContent 解析飞书消息内容（JSON 格式）
func ParseMessageContent(content string) (string, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("JSON 解析失败: %w", err)
	}

	// 提取 text 字段
	if text, ok := data["text"].(string); ok && text != "" {
		return text, nil
	}

	return content, nil
}

// ExtractChatType 从事件中提取会话类型 (group / p2p)
func ExtractChatType(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if chatType, ok := message["chat_type"].(string); ok {
				return chatType, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 chat_type")
}

// ExtractSenderChatID 从事件中提取会话 ID (chat_id)
func ExtractSenderChatID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if chatID, ok := message["chat_id"].(string); ok {
				return chatID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 chat_id")
}
