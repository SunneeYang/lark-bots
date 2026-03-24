package matcher

import "context"

type CascadeMatcher struct {
	matchers []TaskMatcher
}

func NewCascadeMatcher(matchers ...TaskMatcher) *CascadeMatcher {
	return &CascadeMatcher{matchers: matchers}
}

func (c *CascadeMatcher) Match(ctx context.Context, userInput string, candidates []string) (MatchResult, error) {
	for _, m := range c.matchers {
		result, err := m.Match(ctx, userInput, candidates)
		if err != nil {
			continue // 当前 matcher 失败，尝试下一个
		}
		if result.Confidence > 0 {
			return result, nil
		}
	}
	return MatchResult{Confidence: 0}, nil
}
