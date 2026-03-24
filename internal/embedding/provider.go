package embedding

import "context"

// Provider Embedding 生成接口
type Provider interface {
	// Embed 生成文本向量，返回向量数组，每个向量维度由具体实现决定
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// Name 返回 provider 名称
	Name() string
}
