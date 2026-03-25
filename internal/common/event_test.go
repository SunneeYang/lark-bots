package common

import (
	"testing"
)

func TestExtractMessageID(t *testing.T) {
	tests := []struct {
		name    string
		event   interface{}
		want    string
		wantErr bool
	}{
		{
			name: "正常事件",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"message_id": "om_test123",
				},
			},
			want:    "om_test123",
			wantErr: false,
		},
		{
			name: "缺少 message 字段",
			event: map[string]interface{}{
				"sender": map[string]interface{}{
					"user_id": "test_user",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "缺少 message_id 字段",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"chat_id": "oc_test",
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name:    "非 map 类型",
			event:   "invalid",
			want:    "",
			wantErr: true,
		},
		{
			name: "message 不是 map 类型",
			event: map[string]interface{}{
				"message": "invalid",
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractMessageID(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractMessageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExtractMessageID() = %v, want %v", got, tt.want)
			}
		})
	}
}
