package matcher

import (
	"context"
	"testing"
)

// MockProvider 用于测试 EmbeddingMatcher
type mockProvider struct {
	embeddings [][]float64
	err        error
	name       string
}

func (m *mockProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.embeddings, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func TestEmbeddingMatcher_Match(t *testing.T) {
	tests := []struct {
		name       string
		userInput  string
		candidates []string
		provider   *mockProvider
		threshold  float64
		want       string
		wantConf   float64
	}{
		{
			name:       "完全匹配",
			userInput:  "deploy",
			candidates: []string{"deploy", "backup"},
			provider: &mockProvider{
				embeddings: [][]float64{
					{1, 0, 0}, // deploy
					{1, 0, 0}, // deploy
					{0, 1, 0}, // backup
				},
			},
			threshold: 0.7,
			want:      "deploy",
			wantConf:  1.0,
		},
		{
			name:       "低于阈值",
			userInput:  "xxx",
			candidates: []string{"deploy", "backup"},
			provider: &mockProvider{
				embeddings: [][]float64{
					{0, 0, 1}, // xxx (正交)
					{1, 0, 0}, // deploy
					{0, 1, 0}, // backup
				},
			},
			threshold: 0.7,
			want:      "",
			wantConf:  0.0,
		},
		{
			name:       "空候选列表",
			userInput:  "deploy",
			candidates: []string{},
			provider: &mockProvider{
				embeddings: [][]float64{},
			},
			threshold: 0.7,
			want:      "",
			wantConf:  0.0,
		},
		{
			name:       "部分相似",
			userInput:  "部署",
			candidates: []string{"deploy", "backup"},
			provider: &mockProvider{
				embeddings: [][]float64{
					{0.5, 0.5, 0}, // 部署 (中等相似度)
					{1, 0, 0},     // deploy
					{0, 1, 0},     // backup
				},
			},
			threshold: 0.4,
			want:      "deploy",
			wantConf:  0.7071067811865475, // sqrt(0.5)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewEmbeddingMatcher(tt.provider, tt.threshold)
			result, err := m.Match(context.Background(), tt.userInput, tt.candidates)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Matched != tt.want {
				t.Errorf("Matched = %v, want %v", result.Matched, tt.want)
			}
			if result.Confidence != tt.wantConf {
				t.Errorf("Confidence = %v, want %v", result.Confidence, tt.wantConf)
			}
			if tt.want != "" && result.Method != "semantic" {
				t.Errorf("Method = %v, want semantic", result.Method)
			}
		})
	}
}

func TestEmbeddingMatcher_DefaultThreshold(t *testing.T) {
	provider := &mockProvider{
		embeddings: [][]float64{
			{1, 0, 0},
			{1, 0, 0},
		},
	}

	// 阈值为 0 时应使用默认值 0.7
	m := NewEmbeddingMatcher(provider, 0)
	if m.threshold != 0.7 {
		t.Errorf("threshold = %v, want 0.7", m.threshold)
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    []float64
		b    []float64
		want float64
	}{
		{
			name: "相同向量",
			a:    []float64{1, 0, 0},
			b:    []float64{1, 0, 0},
			want: 1.0,
		},
		{
			name: "正交向量",
			a:    []float64{1, 0, 0},
			b:    []float64{0, 1, 0},
			want: 0.0,
		},
		{
			name: "相反向量",
			a:    []float64{1, 0, 0},
			b:    []float64{-1, 0, 0},
			want: -1.0,
		},
		{
			name: "不同长度",
			a:    []float64{1, 0},
			b:    []float64{1, 0, 0},
			want: 0.0,
		},
		{
			name: "空向量",
			a:    []float64{},
			b:    []float64{},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}
