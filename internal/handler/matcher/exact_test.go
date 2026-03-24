package matcher

import (
	"context"
	"testing"
)

func TestExactMatcher_Match(t *testing.T) {
	m := NewExactMatcher()
	ctx := context.Background()
	candidates := []string{"deploy", "backup", "health_check"}

	tests := []struct {
		name       string
		input      string
		want       string
		confidence float64
	}{
		{
			name:       "完全匹配",
			input:      "deploy",
			want:       "deploy",
			confidence: 1.0,
		},
		{
			name:       "不匹配",
			input:      "发布",
			want:       "",
			confidence: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := m.Match(ctx, tt.input, candidates)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Matched != tt.want {
				t.Errorf("Matched = %q, want %q", result.Matched, tt.want)
			}
			if result.Confidence != tt.confidence {
				t.Errorf("Confidence = %v, want %v", result.Confidence, tt.confidence)
			}
			// 只在匹配成功时检查 Method
			if tt.confidence > 0 && result.Method != "exact" {
				t.Errorf("Method = %q, want %q", result.Method, "exact")
			}
		})
	}
}
