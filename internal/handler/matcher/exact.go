package matcher

import "context"

type ExactMatcher struct{}

func NewExactMatcher() *ExactMatcher {
	return &ExactMatcher{}
}

func (m *ExactMatcher) Match(ctx context.Context, userInput string, candidates []string) (MatchResult, error) {
	for _, candidate := range candidates {
		if userInput == candidate {
			return MatchResult{
				Matched:    candidate,
				Confidence: 1.0,
				Method:     "exact",
			}, nil
		}
	}
	return MatchResult{Confidence: 0}, nil
}
