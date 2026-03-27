package matcher

import (
	"context"
	"fmt"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/embedding"
)

// TaskNames 操作名配置（支持别名/同义词）
type TaskNames struct {
	// Primary 主操作名（如"重启"），作为执行时的标准操作标识
	Primary string
	// Aliases 同义词/别名列表（如"重新启动"、"restart"）
	// 用户输入精确或语义匹配到任意一个都算命中
	Aliases []string
}

// ExecutorTaskConfig 任务配置（配置解析后的内部格式）
type ExecutorTaskConfig struct {
	// ExecutorID 执行机器人 ID（用于日志和消息路由）
	ExecutorID string
	// ExecutorName 执行机器人名称
	ExecutorName string
	// Description 执行机器人描述
	Description string
	// RoutingKeywords 路由关键词列表（第一层粗匹配）
	RoutingKeywords []string
	// Keywords 任务关键词列表（第二层精匹配）
	Keywords []string
	// TaskName 唯一任务名称（全局唯一，用于 dispatcher 发送和 executor 认领）
	TaskName string
	// DisplayName 可读性好的任务显示名称（用于用户反馈）
	DisplayName string
	// Script 对应脚本路径
	Script string
	// Names 操作名配置（第三层匹配）
	Names TaskNames
}

// LayeredMatcher 分层任务匹配器
// 三层过滤：routing_keywords(粗) → keywords(精) → names(语义兜底)
type LayeredMatcher struct {
	executors     []ExecutorTaskConfig
	semanticMatch *EmbeddingMatcher
	negationWords []string
}

// NewLayeredMatcher 创建分层匹配器
// executors 从所有 executor 的 tasks 扁平化构建
// semanticProvider 为 nil 时不启用语义匹配
func NewLayeredMatcher(executors []ExecutorTaskConfig, semanticProvider embedding.Provider) *LayeredMatcher {
	return &LayeredMatcher{
		executors:     executors,
		semanticMatch: NewEmbeddingMatcherWithProvider(semanticProvider, 0.7),
		negationWords: []string{"不要", "别", "don’t", "don't", "stop", "禁止", "取消", "不重启", "不要重启"},
	}
}

// NewEmbeddingMatcherWithProvider 创建带 provider 的语义匹配器
func NewEmbeddingMatcherWithProvider(provider embedding.Provider, threshold float64) *EmbeddingMatcher {
	if provider == nil {
		return nil
	}
	return NewEmbeddingMatcher(provider, threshold)
}

// LayeredMatchInput 匹配输入
type LayeredMatchInput struct {
	UserInput string
}

// LayeredMatchResult 匹配结果
type LayeredMatchResult struct {
	ExecutorID   string // 匹配的执行机器人 ID
	ExecutorName string
	TaskName     string // 唯一任务名称（用于 dispatcher 发送）
	DisplayName  string // 可读性好的任务显示名称（用于用户反馈）
	Script       string // 脚本路径（仅 executor 本地使用）
	Operation    string // 命中的操作名（如"重启"）
	Confidence   float64
	Method       string // "exact" | "semantic"
	Error        error
}

// LayeredMatch 所有命中的结果（用于多意图检测）
type LayeredMatchAll struct {
	Matches     []LayeredMatchResult
	HasNegation bool // 输入中是否包含否定词
}

// Match 执行分层匹配
func (m *LayeredMatcher) Match(ctx context.Context, input LayeredMatchInput) LayeredMatchAll {
	userInput := strings.TrimSpace(input.UserInput)

	// Step 0: 否定检测（如果检测到否定意图，直接返回空匹配）
	hasNegation := m.containsNegation(userInput)
	if hasNegation {
		return LayeredMatchAll{
			Matches:     []LayeredMatchResult{}, // 否定意图时不返回任何匹配
			HasNegation: true,
		}
	}

	// 第一轮：只做精确匹配，不启用语义匹配
	var exactMatches []LayeredMatchResult

	// 遍历所有 executor-task 组合
	for _, executor := range m.executors {
		// ===== 第一层：Executor 路由（粗粒度模糊匹配）=====
		if !m.matchRoutingKeywords(userInput, executor.RoutingKeywords) {
			continue
		}
		stripped := m.stripRoutingKeywords(userInput, executor.RoutingKeywords)

		// ===== 第二层：Task 关键词过滤（精匹配）=====
		matchedKeyword := m.matchKeywords(stripped, executor.Keywords)
		if matchedKeyword == "" {
			continue
		}
		remaining := m.stripKeyword(stripped, executor.Keywords)

		// ===== 第三层：操作名精确匹配（不启用语义）=====
		operation, method, confidence := m.matchOperationExactOnly(remaining, executor.Names)
		if operation == "" {
			continue // 操作匹配失败
		}

		// DEBUG: 打印匹配详情
		fmt.Printf("🔍 [Matcher] TaskName=%s, routingKeywords=%v, keywords=%v, names=%v\n",
			executor.TaskName, executor.RoutingKeywords, executor.Keywords, executor.Names.Aliases)
		fmt.Printf("   用户输入: '%s'\n", userInput)
		fmt.Printf("   剥离routing后: '%s'\n", stripped)
		fmt.Printf("   剥离keyword后: '%s'\n", remaining)
		fmt.Printf("   匹配到操作: '%s' (method=%s, confidence=%.2f)\n", operation, method, confidence)

		exactMatches = append(exactMatches, LayeredMatchResult{
			ExecutorID:   executor.ExecutorID,
			ExecutorName: executor.ExecutorName,
			TaskName:     executor.TaskName,
			DisplayName:  executor.DisplayName,
			Script:       executor.Script,
			Operation:    operation,
			Confidence:   confidence,
			Method:       method,
		})
	}

	// 如果有精确匹配，直接返回，不启用语义匹配
	if len(exactMatches) > 0 {
		return LayeredMatchAll{
			Matches:     exactMatches,
			HasNegation: false,
		}
	}

	// 第二轮：没有精确匹配，才启用语义匹配
	fmt.Println("🔍 [Matcher] 没有精确匹配，启用语义匹配...")
	var semanticMatches []LayeredMatchResult

	for _, executor := range m.executors {
		// ===== 第一层：Executor 路由（粗粒度模糊匹配）=====
		if !m.matchRoutingKeywords(userInput, executor.RoutingKeywords) {
			continue
		}
		stripped := m.stripRoutingKeywords(userInput, executor.RoutingKeywords)

		// ===== 第二层：Task 关键词过滤（精匹配）=====
		matchedKeyword := m.matchKeywords(stripped, executor.Keywords)
		if matchedKeyword == "" {
			continue
		}
		remaining := m.stripKeyword(stripped, executor.Keywords)

		// ===== 第三层：操作名匹配（启用语义兜底）=====
		operation, method, confidence := m.matchOperation(remaining, executor.Names)
		if operation == "" {
			continue
		}

		fmt.Printf("🔍 [Matcher] TaskName=%s, routingKeywords=%v, keywords=%v, names=%v\n",
			executor.TaskName, executor.RoutingKeywords, executor.Keywords, executor.Names.Aliases)
		fmt.Printf("   用户输入: '%s'\n", userInput)
		fmt.Printf("   剥离routing后: '%s'\n", stripped)
		fmt.Printf("   剥离keyword后: '%s'\n", remaining)
		fmt.Printf("   匹配到操作: '%s' (method=%s, confidence=%.2f)\n", operation, method, confidence)

		semanticMatches = append(semanticMatches, LayeredMatchResult{
			ExecutorID:   executor.ExecutorID,
			ExecutorName: executor.ExecutorName,
			TaskName:     executor.TaskName,
			DisplayName:  executor.DisplayName,
			Script:       executor.Script,
			Operation:    operation,
			Confidence:   confidence,
			Method:       method,
		})
	}

	return LayeredMatchAll{
		Matches:     semanticMatches,
		HasNegation: false,
	}
}

// matchRoutingKeywords 第一层：判断输入是否匹配 executor 的路由关键词
func (m *LayeredMatcher) matchRoutingKeywords(input string, routingKeywords []string) bool {
	if len(routingKeywords) == 0 {
		return true // 无 routing_keywords 时默认匹配（兼容旧配置）
	}
	input = strings.ToLower(input)
	for _, kw := range routingKeywords {
		if strings.Contains(input, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// stripRoutingKeywords 第一层：剥离已匹配的 routing keywords
func (m *LayeredMatcher) stripRoutingKeywords(input string, routingKeywords []string) string {
	if len(routingKeywords) == 0 {
		return input
	}
	result := input
	for _, kw := range routingKeywords {
		result = m.removeSubstring(result, kw)
	}
	return strings.TrimSpace(result)
}

// matchKeywords 第二层：匹配 task 关键词
func (m *LayeredMatcher) matchKeywords(input string, keywords []string) string {
	input = strings.ToLower(input)
	for _, kw := range keywords {
		if strings.Contains(input, strings.ToLower(kw)) {
			return kw
		}
	}
	return ""
}

// stripKeyword 第二层：剥离已匹配的 keyword
func (m *LayeredMatcher) stripKeyword(input string, keywords []string) string {
	result := input
	for _, kw := range keywords {
		result = m.removeSubstring(result, kw)
	}
	return strings.TrimSpace(result)
}

// matchOperation 第三层：匹配操作名（精确优先，语义兜底）
func (m *LayeredMatcher) matchOperation(remaining string, names TaskNames) (operation, method string, confidence float64) {
	// 优先精确匹配 primary 和 aliases
	lowerRemaining := strings.ToLower(remaining)
	for _, alias := range names.Aliases {
		if strings.Contains(lowerRemaining, strings.ToLower(alias)) {
			return names.Primary, "exact", 1.0
		}
	}
	// 精确匹配 primary 本身
	if strings.Contains(lowerRemaining, strings.ToLower(names.Primary)) {
		return names.Primary, "exact", 1.0
	}

	// 语义兜底（仅当 remaining 非空且语义匹配器可用时）
	if remaining != "" && m.semanticMatch != nil {
		candidates := append([]string{names.Primary}, names.Aliases...)
		result, err := m.semanticMatch.Match(ctxWithoutCancel(), remaining, candidates)
		if err == nil && result.Confidence > 0 {
			return result.Matched, result.Method, result.Confidence
		}
	}

	return "", "", 0
}

// matchOperationExactOnly 第三层：仅做精确匹配，不启用语义匹配
func (m *LayeredMatcher) matchOperationExactOnly(remaining string, names TaskNames) (operation, method string, confidence float64) {
	lowerRemaining := strings.ToLower(remaining)

	// 精确匹配 aliases
	for _, alias := range names.Aliases {
		if strings.Contains(lowerRemaining, strings.ToLower(alias)) {
			return names.Primary, "exact", 1.0
		}
	}

	// 精确匹配 primary 本身
	if strings.Contains(lowerRemaining, strings.ToLower(names.Primary)) {
		return names.Primary, "exact", 1.0
	}

	// 不启用语义匹配，直接返回空
	return "", "", 0
}

// containsNegation 检测输入是否包含否定词
func (m *LayeredMatcher) containsNegation(input string) bool {
	lower := strings.ToLower(input)
	for _, neg := range m.negationWords {
		if strings.Contains(lower, strings.ToLower(neg)) {
			return true
		}
	}
	return false
}

// removeSubstring 移除字符串中所有匹配的子串（大小写不敏感），返回标准化后的剩余文本
func (m *LayeredMatcher) removeSubstring(input, toRemove string) string {
	lower := strings.ToLower(input)
	removeLower := strings.ToLower(toRemove)
	result := strings.ReplaceAll(lower, removeLower, "")
	// 规范化空白字符（移除多余空格）
	return strings.Join(strings.Fields(result), " ")
}

// ctxWithoutCancel 返回一个不被 context 取消的子 context
func ctxWithoutCancel() context.Context {
	return context.WithoutCancel(context.Background())
}
