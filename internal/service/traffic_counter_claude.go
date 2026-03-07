package service

import (
	"fmt"
	"log"
	"regexp"
)

// 本文件为 Claude 专用流量与计费封装，基于 Anthropic 官方定价实现语义化接口

// ClaudeModel 定义 Claude 模型类型
type ClaudeModel string

const (
	ClaudeModelHaiku45  ClaudeModel = "haiku-4.5"  // Claude Haiku 4.5
	ClaudeModelSonnet45 ClaudeModel = "sonnet-4.5" // Claude Sonnet 4.5
	ClaudeModelOpus45   ClaudeModel = "opus-4.5"   // Claude Opus 4.5
	ClaudeModelOpus46   ClaudeModel = "opus-4.6"   // Claude Opus 4.6 (2026年2月发布，定价同 Opus 4.5)
)

// DetectClaudeModel 从模型字符串检测 Claude 模型类型
// 支持的格式: "claude-haiku-4-5-xxx", "claude-sonnet-4-5-xxx", "claude-opus-4-5-xxx", "claude-opus-4-6-xxx"
// 也支持 JSON 格式: "model":"claude-sonnet-4-5-20250929"
// 通过正则提取 claude- 后的模型名称，兼容未来新模型（如 claude-max、claude-plus 等）
func DetectClaudeModel(modelStr string) ClaudeModel {
	if modelStr == "" {
		return ClaudeModelSonnet45 // 默认 Sonnet 4.5
	}

	// 先尝试从 JSON 格式中提取完整模型名称: "model":"claude-xxx-xxx"
	// 例如: "model":"claude-sonnet-4-5-20250929" -> "claude-sonnet-4-5-20250929"
	fullModelPattern := regexp.MustCompile(`(?i)"model"\s*:\s*"(claude-[^"]+)"`)
	fullMatches := fullModelPattern.FindStringSubmatch(modelStr)

	var extractedModel string
	if len(fullMatches) >= 2 {
		extractedModel = fullMatches[1]
		log.Printf("[调试] 从 JSON 提取完整模型名称: %s", extractedModel)
	} else {
		extractedModel = modelStr
	}

	// 检测是否为 Opus 4.6（优先检测版本号）
	if regexp.MustCompile(`(?i)opus.*4[-_.]6`).MatchString(extractedModel) {
		return ClaudeModelOpus46
	}

	// 提取 claude- 后面的第一个单词（模型类型）
	// 例如: "claude-sonnet-4-5-20250929" -> "sonnet"
	//      "claude-max-5-0" -> "max"
	modelPattern := regexp.MustCompile(`(?i)claude[-_.\s]([a-z]+)`)
	matches := modelPattern.FindStringSubmatch(extractedModel)

	if len(matches) < 2 {
		// 如果没有匹配到，尝试直接匹配关键词
		if regexp.MustCompile(`(?i)haiku`).MatchString(modelStr) {
			return ClaudeModelHaiku45
		}
		if regexp.MustCompile(`(?i)opus`).MatchString(modelStr) {
			return ClaudeModelOpus45
		}
		if regexp.MustCompile(`(?i)sonnet`).MatchString(modelStr) {
			return ClaudeModelSonnet45
		}
		return ClaudeModelSonnet45 // 默认
	}

	// 提取到的模型名称（转小写）
	modelName := matches[1]

	// 根据模型名称匹配
	switch modelName {
	case "haiku":
		// Haiku 目前只有 4.5 版本
		return ClaudeModelHaiku45
	case "opus":
		// Opus 默认返回 4.5，4.6 已在上面优先检测
		return ClaudeModelOpus45
	case "sonnet":
		// Sonnet 目前只有 4.5 版本
		return ClaudeModelSonnet45
	default:
		// 未知模型，默认使用 Sonnet 4.5 计费
		log.Printf("[警告] 未知的 Claude 模型: %s，使用 Sonnet 4.5 计费", modelName)
		return ClaudeModelSonnet45
	}
}

// ClaudePricingConfig Claude 计费配置
type ClaudePricingConfig struct {
	Model          ClaudeModel // 模型类型
	UseBatchAPI    bool        // 是否使用 Batch API（享受 50% 折扣）
	CachedHitRatio float64     // 缓存命中率（0.0-1.0）
	InputRatio     float64     // 总 tokens 中输入占比（默认 0.70）
	OutputRatio    float64     // 总 tokens 中输出占比（默认 0.30）
}

// DefaultClaudePricingConfig 返回默认配置（Sonnet 4.5，不使用 Batch API，10% 缓存命中）
func DefaultClaudePricingConfig() ClaudePricingConfig {
	return ClaudePricingConfig{
		Model:          ClaudeModelSonnet45,
		UseBatchAPI:    false,
		CachedHitRatio: 0.10,
		InputRatio:     0.70,
		OutputRatio:    0.30,
	}
}

// NewClaudeTrafficCounter 初始化 Claude 流量计数器（复用通用实现）
func NewClaudeTrafficCounter(customerToken string) *TrafficCounter {
	// 记录用户访问
	trackingService := NewUserTrackingService()
	if err := trackingService.TrackUserAccess(customerToken); err != nil {
		log.Printf("[警告] 记录用户访问失败: %v", err)
	}
	return NewTrafficCounter(customerToken)
}

// SendClaudeTokenUsageToStreamWithIO 发送 Claude token 使用量（含方向和模型）到 Redis Stream
func SendClaudeTokenUsageToStreamWithIO(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, model *string) error {
	return SendTokenUsageToStreamWithIOAndModel(customerToken, tokens, inTokens, outTokens, model)
}

// SendClaudeTokenUsageToStreamWithIOAndMessage 发送 Claude token + 原始消息体到 Redis Stream
func SendClaudeTokenUsageToStreamWithIOAndMessage(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, model *string, originMessage *string) error {
	return SendTokenUsageToStreamWithIOAndModelAndMessage(customerToken, tokens, inTokens, outTokens, model, originMessage)
}

// claudePricing Claude 定价结构（内部使用）
type claudePricing struct {
	InputPerM       float64 // 标准输入价格（每百万 tokens）
	OutputPerM      float64 // 输出价格（每百万 tokens）
	CachedInputPerM float64 // 缓存读取价格（每百万 tokens）
}

// getClaudePricing 获取 Claude 定价（支持 Batch API 折扣）
// 定价规则:
// - Haiku 4.5: 输入 $1/M, 输出 $5/M, 缓存 $0.10/M
// - Sonnet 4.5: 输入 $3/M, 输出 $15/M, 缓存 $0.30/M
// - Opus 4.5: 输入 $5/M, 输出 $25/M, 缓存 $0.50/M (Opus = 5×Haiku)
// - 缓存读取享受 10 倍折扣
// - Batch API 享受 50% 折扣
func getClaudePricing(model ClaudeModel, useBatchAPI bool) claudePricing {
	var prices claudePricing

	switch model {
	case ClaudeModelHaiku45:
		// Claude Haiku 4.5 定价（最经济）
		prices = claudePricing{
			InputPerM:       1.00, // $1 / M input
			OutputPerM:      5.00, // $5 / M output
			CachedInputPerM: 0.10, // $0.10 / M cached read (10倍折扣)
		}
	case ClaudeModelSonnet45:
		// Claude Sonnet 4.5 定价（平衡）
		prices = claudePricing{
			InputPerM:       3.00,  // $3 / M input
			OutputPerM:      15.00, // $15 / M output
			CachedInputPerM: 0.30,  // $0.30 / M cached read (10倍折扣)
		}
	case ClaudeModelOpus45:
		// Claude Opus 4.5 定价（最强大，价格是 Haiku 的 5 倍）
		prices = claudePricing{
			InputPerM:       5.00,  // $5 / M input (5×Haiku)
			OutputPerM:      25.00, // $25 / M output (5×Haiku)
			CachedInputPerM: 0.50,  // $0.50 / M cached read (10倍折扣)
		}
	default:
		// 默认回退到 Sonnet 4.5
		prices = claudePricing{
			InputPerM:       3.00,
			OutputPerM:      15.00,
			CachedInputPerM: 0.30,
		}
	}

	// Batch API 50% 折扣（所有价格减半）
	if useBatchAPI {
		prices.InputPerM *= 0.5       // 例如 Opus: $5 → $2.50
		prices.OutputPerM *= 0.5      // 例如 Opus: $25 → $12.50
		prices.CachedInputPerM *= 0.5 // 例如 Opus: $0.50 → $0.25
	}

	return prices
}

// ClaudeTokensToConsumeByIOWithConfig 按输入/输出分别计费（支持自定义配置）
func ClaudeTokensToConsumeByIOWithConfig(inTokens uint64, outTokens uint64, config ClaudePricingConfig) float64 {
	if inTokens == 0 && outTokens == 0 {
		return 0
	}

	prices := getClaudePricing(config.Model, config.UseBatchAPI)

	// 计算缓存命中部分
	cachedInputTokens := uint64(float64(inTokens) * config.CachedHitRatio)
	normalInputTokens := inTokens - cachedInputTokens

	inputCost := (float64(normalInputTokens)*prices.InputPerM + float64(cachedInputTokens)*prices.CachedInputPerM) / 1_000_000.0
	outputCost := float64(outTokens) * prices.OutputPerM / 1_000_000.0

	return inputCost + outputCost
}

// ClaudeTokensToConsumeByIO 计算 Claude 的计费（按输入/输出，使用默认配置）
func ClaudeTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, DefaultClaudePricingConfig())
}

// ClaudeTokensToConsumeWithConfig 按总 tokens 计费（按比例拆分输入/输出，支持自定义配置）
func ClaudeTokensToConsumeWithConfig(tokens uint64, config ClaudePricingConfig) float64 {
	if tokens == 0 {
		return 0
	}

	inTokens := uint64(float64(tokens) * config.InputRatio)
	outTokens := uint64(float64(tokens) * config.OutputRatio)

	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
}

// ClaudeTokensToConsume 计算 Claude 的计费（总 tokens，使用默认配置）
func ClaudeTokensToConsume(tokens uint64) float64 {
	return ClaudeTokensToConsumeWithConfig(tokens, DefaultClaudePricingConfig())
}

// ClaudeHaikuTokensToConsumeByIO Haiku 4.5 专用计费（按输入/输出）
func ClaudeHaikuTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	config := DefaultClaudePricingConfig()
	config.Model = ClaudeModelHaiku45
	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
}

// ClaudeSonnetTokensToConsumeByIO Sonnet 4.5 专用计费（按输入/输出）
func ClaudeSonnetTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	config := DefaultClaudePricingConfig()
	config.Model = ClaudeModelSonnet45
	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
}

// ClaudeOpusTokensToConsumeByIO Opus 4.5 专用计费（按输入/输出）
// 定价: 输入 $5/M, 输出 $25/M, 缓存 $0.50/M
func ClaudeOpusTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	config := DefaultClaudePricingConfig()
	config.Model = ClaudeModelOpus45
	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
}

// ========== Batch API 专用计费函数（享受 50% 折扣）==========

// ClaudeBatchTokensToConsumeByIO Batch API 专用计费（按输入/输出，50% 折扣）
func ClaudeBatchTokensToConsumeByIO(inTokens uint64, outTokens uint64, model ClaudeModel) float64 {
	config := DefaultClaudePricingConfig()
	config.Model = model
	config.UseBatchAPI = true
	return ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
}

// ClaudeHaikuBatchTokensToConsumeByIO Haiku 4.5 Batch API 计费
// 定价: 输入 $0.50/M, 输出 $2.50/M, 缓存 $0.05/M
func ClaudeHaikuBatchTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	return ClaudeBatchTokensToConsumeByIO(inTokens, outTokens, ClaudeModelHaiku45)
}

// ClaudeSonnetBatchTokensToConsumeByIO Sonnet 4.5 Batch API 计费
// 定价: 输入 $1.50/M, 输出 $7.50/M, 缓存 $0.15/M
func ClaudeSonnetBatchTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	return ClaudeBatchTokensToConsumeByIO(inTokens, outTokens, ClaudeModelSonnet45)
}

// ClaudeOpusBatchTokensToConsumeByIO Opus 4.5 Batch API 计费
// 定价: 输入 $2.50/M, 输出 $12.50/M, 缓存 $0.25/M
func ClaudeOpusBatchTokensToConsumeByIO(inTokens uint64, outTokens uint64) float64 {
	return ClaudeBatchTokensToConsumeByIO(inTokens, outTokens, ClaudeModelOpus45)
}

// WriteClaudeToMySQL 写入 MySQL（Claude 定价）
// 根据是否提供 in/out 拆分来计算更精确的消费金额
// model: 可选的模型名称（例如 "claude-haiku-4-5-20251001"）
func WriteClaudeToMySQL(customerToken string, tokens uint64, inTokens uint64, outTokens uint64, hash string, model *string, originMessage *string) error {
	pricingService := NewPricingService()
	consume, pricingRule, accountType, calcErr := pricingService.CalculateConsumeByCustomerToken(customerToken, tokens, inTokens, outTokens)
	if calcErr != nil {
		// 兼容回退：保留原 Claude 模型计价
		config := DefaultClaudePricingConfig()
		if model != nil {
			config.Model = DetectClaudeModel(*model)
		}
		if inTokens > 0 || outTokens > 0 {
			consume = ClaudeTokensToConsumeByIOWithConfig(inTokens, outTokens, config)
		} else {
			consume = ClaudeTokensToConsumeWithConfig(tokens, config)
		}
		pricingRule = nil
		log.Printf("[计价/Claude] 查询 pricing 失败，回退 Claude 内置模型计价: token=%s, error=%v", maskKey(customerToken), calcErr)
	}

	usageService := NewUsageService()
	if err := usageService.RecordTokenUsage(customerToken, tokens, consume, model, originMessage); err != nil {
		return fmt.Errorf("failed to record token usage: %w", err)
	}

	// 记录 token 使用量到限流器
	ConsumeTokens(customerToken, "claude", int(tokens))

	modelStr := "unknown"
	if model != nil {
		modelStr = *model
	}

	unit := DefaultPricingUnit
	if pricingRule != nil && pricingRule.Unit != "" {
		unit = pricingRule.Unit
	}
	log.Printf("[MySQL/Claude] 写入成功: token=%s, tokens=%d, consume=%.6f (%s), account_type=%s, in=%d, out=%d, model=%s, hash=%s",
		maskKey(customerToken), tokens, consume, unit, accountType, inTokens, outTokens, modelStr, hash)
	return nil
}
