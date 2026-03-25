package matcher

import (
	"context"
	"testing"
)

// TestLayeredMatcher_RoutingKeywords 测试第一层路由匹配
func TestLayeredMatcher_RoutingKeywords(t *testing.T) {
	executors := []ExecutorTaskConfig{
		{
			ExecutorID:      "dev-executor",
			ExecutorName:    "dev-executor",
			RoutingKeywords: []string{"开发", "dev"},
			Description:     "开发服执行器",
		},
		{
			ExecutorID:      "test-executor",
			ExecutorName:    "test-executor",
			RoutingKeywords: []string{"测试", "test"},
			Description:     "测试服执行器",
		},
	}

	matcher := NewLayeredMatcher(executors, nil)

	tests := []struct {
		name     string
		input    string
		wantDev  bool
		wantTest bool
	}{
		{
			name:    "匹配开发关键词",
			input:   "土豆开发服重启",
			wantDev: true,
		},
		{
			name:     "匹配测试关键词",
			input:    "迷雾测试服更新",
			wantTest: true,
		},
		{
			name:  "无匹配关键词",
			input: "生产服重启",
		},
		{
			name:    "匹配dev缩写",
			input:   "土豆dev重启",
			wantDev: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 测试路由关键词匹配
			devMatched := matcher.matchRoutingKeywords(tt.input, executors[0].RoutingKeywords)
			testMatched := matcher.matchRoutingKeywords(tt.input, executors[1].RoutingKeywords)

			if tt.wantDev && !devMatched {
				t.Errorf("期望匹配 dev，但未匹配")
			}
			if tt.wantTest && !testMatched {
				t.Errorf("期望匹配 test，但未匹配")
			}
			if !tt.wantDev && devMatched {
				t.Errorf("不期望匹配 dev，但匹配了")
			}
			if !tt.wantTest && testMatched {
				t.Errorf("不期望匹配 test，但匹配了")
			}
		})
	}
}

// TestLayeredMatcher_Keywords 测试第二层关键词匹配
func TestLayeredMatcher_Keywords(t *testing.T) {
	matcher := &LayeredMatcher{}

	tests := []struct {
		name     string
		input    string
		keywords []string
		want     string
	}{
		{
			name:     "精确匹配关键词",
			input:    "土豆开发服重启",
			keywords: []string{"土豆", "迷雾"},
			want:     "土豆",
		},
		{
			name:     "无匹配关键词",
			input:    "南瓜开发服重启",
			keywords: []string{"土豆", "迷雾"},
			want:     "",
		},
		{
			name:     "大小写不敏感匹配",
			input:    "POTATO开发服重启",
			keywords: []string{"potato"},
			want:     "potato",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.matchKeywords(tt.input, tt.keywords)
			if got != tt.want {
				t.Errorf("matchKeywords() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLayeredMatcher_Operation 测试第三层操作匹配
func TestLayeredMatcher_Operation(t *testing.T) {
	matcher := NewLayeredMatcher(nil, nil)

	tests := []struct {
		name      string
		remaining string
		names     TaskNames
		wantOp    string
		wantMethod string
	}{
		{
			name:      "精确匹配主操作名",
			remaining: "重启",
			names: TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "重新启动"},
			},
			wantOp:    "重启",
			wantMethod: "exact",
		},
		{
			name:      "精确匹配别名",
			remaining: "重新启动",
			names: TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "重新启动", "restart"},
			},
			wantOp:    "重启",
			wantMethod: "exact",
		},
		{
			name:      "无匹配",
			remaining: "部署",
			names: TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "重新启动"},
			},
			wantOp:     "",
			wantMethod: "",
		},
		{
			name:      "部分匹配操作名",
			remaining: "重启一下",
			names: TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "重新启动"},
			},
			wantOp:    "重启",
			wantMethod: "exact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOp, gotMethod, _ := matcher.matchOperation(tt.remaining, tt.names)
			if gotOp != tt.wantOp {
				t.Errorf("matchOperation() op = %v, want %v", gotOp, tt.wantOp)
			}
			if gotMethod != tt.wantMethod {
				t.Errorf("matchOperation() method = %v, want %v", gotMethod, tt.wantMethod)
			}
		})
	}
}

// TestLayeredMatcher_StripRoutingKeywords 测试剥离路由关键词
func TestLayeredMatcher_StripRoutingKeywords(t *testing.T) {
	matcher := &LayeredMatcher{}

	tests := []struct {
		name            string
		input           string
		routingKeywords []string
		want            string
	}{
		{
			name:            "剥离单个关键词",
			input:           "土豆开发服重启",
			routingKeywords: []string{"开发"},
			want:            "土豆服重启",
		},
		{
			name:            "剥离多个关键词",
			input:           "土豆dev测试服重启",
			routingKeywords: []string{"dev", "测试"},
			want:            "土豆服重启",
		},
		{
			name:            "无匹配关键词",
			input:           "土豆生产服重启",
			routingKeywords: []string{"开发", "测试"},
			want:            "土豆生产服重启",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.stripRoutingKeywords(tt.input, tt.routingKeywords)
			if got != tt.want {
				t.Errorf("stripRoutingKeywords() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLayeredMatcher_StripKeyword 测试剥离任务关键词
func TestLayeredMatcher_StripKeyword(t *testing.T) {
	matcher := &LayeredMatcher{}

	tests := []struct {
		name     string
		input    string
		keywords []string
		want     string
	}{
		{
			name:     "剥离服务器名",
			input:    "土豆开发服重启",
			keywords: []string{"土豆"},
			want:     "开发服重启",
		},
		{
			name:     "剥离多个关键词",
			input:    "土豆迷雾开发服重启",
			keywords: []string{"土豆", "迷雾"},
			want:     "开发服重启",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.stripKeyword(tt.input, tt.keywords)
			if got != tt.want {
				t.Errorf("stripKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLayeredMatcher_ContainsNegation 测试否定检测
func TestLayeredMatcher_ContainsNegation(t *testing.T) {
	matcher := NewLayeredMatcher(nil, nil)

	tests := []struct {
		name     string
		input    string
		wantBool bool
	}{
		{
			name:     "包含不要",
			input:    "土豆开发服不要重启",
			wantBool: true,
		},
		{
			name:     "包含别",
			input:    "土豆开发服别重启",
			wantBool: true,
		},
		{
			name:     "包含don't",
			input:    "don't restart potato",
			wantBool: true,
		},
		{
			name:     "包含禁止",
			input:    "禁止重启土豆",
			wantBool: true,
		},
		{
			name:     "不包含否定词",
			input:    "土豆开发服重启",
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.containsNegation(tt.input)
			if got != tt.wantBool {
				t.Errorf("containsNegation() = %v, want %v", got, tt.wantBool)
			}
		})
	}
}

// TestLayeredMatcher_Match_EndToEnd 测试完整匹配流程
func TestLayeredMatcher_Match_EndToEnd(t *testing.T) {
	executors := []ExecutorTaskConfig{
		{
			ExecutorID:      "dev-executor-1",
			ExecutorName:    "dev-executor-1",
			Description:     "开发服执行器1",
			RoutingKeywords: []string{"开发", "dev"},
			Keywords:        []string{"土豆"},
			Script:          "/scripts/potato_dev_restart.sh",
			Names: TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "重新启动", "restart"},
			},
		},
		{
			ExecutorID:      "dev-executor-2",
			ExecutorName:    "dev-executor-2",
			Description:     "开发服执行器2",
			RoutingKeywords: []string{"开发", "dev"},
			Keywords:        []string{"迷雾"},
			Script:          "/scripts/mist_dev_update.sh",
			Names: TaskNames{
				Primary: "更新",
				Aliases: []string{"更新", "升级", "update"},
			},
		},
	}

	matcher := NewLayeredMatcher(executors, nil)

	tests := []struct {
		name          string
		input         string
		wantMatches   int
		wantExecutor  string
		wantOperation string
		wantHasNeg    bool
	}{
		{
			name:          "成功匹配土豆重启",
			input:         "土豆开发服重启",
			wantMatches:   1,
			wantExecutor:  "dev-executor-1",
			wantOperation: "重启",
			wantHasNeg:    false,
		},
		{
			name:          "成功匹配迷雾更新",
			input:         "迷雾dev更新",
			wantMatches:   1,
			wantExecutor:  "dev-executor-2",
			wantOperation: "更新",
			wantHasNeg:    false,
		},
		{
			name:       "否定意图",
			input:      "土豆开发服不要重启",
			wantMatches: 0,
			wantHasNeg: true,
		},
		{
			name:       "无匹配",
			input:      "南瓜生产服重启",
			wantMatches: 0,
			wantHasNeg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matcher.Match(context.Background(), LayeredMatchInput{UserInput: tt.input})

			if result.HasNegation != tt.wantHasNeg {
				t.Errorf("HasNegation = %v, want %v", result.HasNegation, tt.wantHasNeg)
			}

			if len(result.Matches) != tt.wantMatches {
				t.Errorf("Matches count = %d, want %d", len(result.Matches), tt.wantMatches)
				return
			}

			if tt.wantMatches > 0 {
				match := result.Matches[0]
				if match.ExecutorID != tt.wantExecutor {
					t.Errorf("ExecutorID = %v, want %v", match.ExecutorID, tt.wantExecutor)
				}
				if match.Operation != tt.wantOperation {
					t.Errorf("Operation = %v, want %v", match.Operation, tt.wantOperation)
				}
			}
		})
	}
}

// TestLayeredMatcher_RemoveSubstring 测试移除子串
func TestLayeredMatcher_RemoveSubstring(t *testing.T) {
	matcher := &LayeredMatcher{}

	tests := []struct {
		name       string
		input      string
		toRemove   string
		wantResult string
	}{
		{
			name:       "移除单个子串",
			input:      "土豆开发服重启",
			toRemove:   "开发",
			wantResult: "土豆服重启",
		},
		{
			name:       "移除多个相同子串",
			input:      "测试测试服重启",
			toRemove:   "测试",
			wantResult: "服重启",
		},
		{
			name:       "大小写不敏感",
			input:      "POTATO开发服重启",
			toRemove:   "potato",
			wantResult: "开发服重启",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matcher.removeSubstring(tt.input, tt.toRemove)
			if got != tt.wantResult {
				t.Errorf("removeSubstring() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
