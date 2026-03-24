package matcher

import (
	"context"
	"math"

	"github.com/SunneeYang/lark-bots/internal/embedding"
)

type EmbeddingMatcher struct {
	provider  embedding.Provider
	threshold float64
}

func NewEmbeddingMatcher(provider embedding.Provider, threshold float64) *EmbeddingMatcher {
	if threshold <= 0 {
		threshold = 0.7
	}
	return &EmbeddingMatcher{
		provider:  provider,
		threshold: threshold,
	}
}

func (m *EmbeddingMatcher) Match(ctx context.Context, userInput string, candidates []string) (MatchResult, error) {
	if len(candidates) == 0 {
		return MatchResult{Confidence: 0}, nil
	}

	// 批量获取 embedding
	allTexts := append([]string{userInput}, candidates...)
	embeddings, err := m.provider.Embed(ctx, allTexts)
	if err != nil {
		return MatchResult{}, err
	}

	// 计算余弦相似度
	userEmbedding := embeddings[0]
	var bestScore float64
	var bestCandidate string

	for i, candidateEmbedding := range embeddings[1:] {
		score := cosineSimilarity(userEmbedding, candidateEmbedding)
		if score > bestScore {
			bestScore = score
			bestCandidate = candidates[i]
		}
	}

	if bestScore >= m.threshold {
		return MatchResult{
			Matched:    bestCandidate,
			Confidence: bestScore,
			Method:     "semantic",
		}, nil
	}

	return MatchResult{Confidence: 0}, nil
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
