package matcher

import "context"

// TaskMatcher 任务匹配器接口
type TaskMatcher interface {
	Match(ctx context.Context, userInput string, candidates []string) (MatchResult, error)
}

// MatchResult 匹配结果
type MatchResult struct {
	Matched    string   // 匹配到的任务名（标准名）
	Confidence float64  // 置信度 0.0-1.0，0 表示未匹配
	Method     string   // "exact" | "semantic"
}
