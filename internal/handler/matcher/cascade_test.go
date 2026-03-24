package matcher

import (
	"context"
	"errors"
	"testing"
)

// mockMatcher 模拟 matcher 用于测试
type mockMatcher struct {
	result MatchResult
	err    error
}

func (m *mockMatcher) Match(ctx context.Context, userInput string, candidates []string) (MatchResult, error) {
	return m.result, m.err
}

func TestCascadeMatcher_Match(t *testing.T) {
	ctx := context.Background()
	candidates := []string{"deploy", "backup"}

	tests := []struct {
		name      string
		matchers  []TaskMatcher
		wantMatch bool
	}{
		{
			name: "第一个匹配成功",
			matchers: []TaskMatcher{
				&mockMatcher{result: MatchResult{Matched: "deploy", Confidence: 1.0}},
			},
			wantMatch: true,
		},
		{
			name: "第一个失败，第二个成功",
			matchers: []TaskMatcher{
				&mockMatcher{err: errors.New("error")},
				&mockMatcher{result: MatchResult{Matched: "backup", Confidence: 0.8}},
			},
			wantMatch: true,
		},
		{
			name: "都失败",
			matchers: []TaskMatcher{
				&mockMatcher{result: MatchResult{}},
				&mockMatcher{result: MatchResult{}},
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCascadeMatcher(tt.matchers...)
			result, _ := c.Match(ctx, "input", candidates)
			hasMatch := result.Confidence > 0
			if hasMatch != tt.wantMatch {
				t.Errorf("Match = %v, want %v", hasMatch, tt.wantMatch)
			}
		})
	}
}
