package service

import (
	"encoding/json"
	"log"
	"regexp"
	"strconv"
)

// ClaudeUsageInfo 从 Claude API 响应中提取的使用信息
type ClaudeUsageInfo struct {
	InputTokens  uint64  // 官方返回的输入 token 数
	OutputTokens uint64  // 官方返回的输出 token 数
	Model        *string // 官方返回的模型名称
}

// ClaudeAPIResponse Claude API 标准响应结构
type ClaudeAPIResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Model      string `json:"model"`
	Role       string `json:"role"`
	StopReason string `json:"stop_reason"`
	Type       string `json:"type"`
	Usage      struct {
		InputTokens  uint64 `json:"input_tokens"`
		OutputTokens uint64 `json:"output_tokens"`
	} `json:"usage"`
}

// ClaudeSSEEvent SSE 事件结构
type ClaudeSSEEvent struct {
	Type  string `json:"type"`
	Usage *struct {
		InputTokens  uint64 `json:"input_tokens"`
		OutputTokens uint64 `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Message *struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  uint64 `json:"input_tokens"`
			OutputTokens uint64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
}

// ParseClaudeResponse 解析 Claude API 响应，提取官方 token 使用量和模型
// 支持标准 JSON 响应和 SSE 流式响应
func ParseClaudeResponse(body []byte, isSSE bool) (*ClaudeUsageInfo, error) {
	if isSSE {
		return parseClaudeSSEResponse(body)
	}
	return parseClaudeJSONResponse(body)
}

// parseClaudeJSONResponse 解析标准 JSON 响应
// 策略1: 尝试 JSON 解析
// 策略2: 正则提取
// 策略3: 根据内容长度估算
func parseClaudeJSONResponse(body []byte) (*ClaudeUsageInfo, error) {
	// 策略1: 尝试标准 JSON 解析
	var resp ClaudeAPIResponse
	if err := json.Unmarshal(body, &resp); err == nil {
		if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
			info := &ClaudeUsageInfo{
				InputTokens:  resp.Usage.InputTokens,
				OutputTokens: resp.Usage.OutputTokens,
			}
			if resp.Model != "" {
				info.Model = &resp.Model
			}
			log.Printf("[Token提取] 使用策略1-JSON解析: input=%d, output=%d, model=%s",
				info.InputTokens, info.OutputTokens, resp.Model)
			return info, nil
		}
	}

	// 策略2: 正则提取 usage
	return extractTokensByRegex(body)
}

// parseClaudeSSEResponse 解析 SSE 流式响应
// 策略1: 尝试逐行 JSON 解析 SSE 事件
// 策略2: 正则提取
// 策略3: 根据内容长度估算
func parseClaudeSSEResponse(body []byte) (*ClaudeUsageInfo, error) {
	var info ClaudeUsageInfo
	var foundUsage bool

	// 策略1: 尝试解析 SSE 事件（逐行解析 data: {...}）
	lines := regexp.MustCompile(`\r?\n`).Split(string(body), -1)
	for _, line := range lines {
		if len(line) > 6 && line[:6] == "data: " {
			jsonData := []byte(line[6:])
			var event ClaudeSSEEvent
			if err := json.Unmarshal(jsonData, &event); err == nil {
				// 从 message_delta 或 message_start 提取 usage
				if event.Usage != nil && (event.Usage.InputTokens > 0 || event.Usage.OutputTokens > 0) {
					info.InputTokens = event.Usage.InputTokens
					info.OutputTokens = event.Usage.OutputTokens
					foundUsage = true
				}
				if event.Message != nil {
					if event.Message.Model != "" {
						info.Model = &event.Message.Model
					}
					if event.Message.Usage.InputTokens > 0 || event.Message.Usage.OutputTokens > 0 {
						info.InputTokens = event.Message.Usage.InputTokens
						info.OutputTokens = event.Message.Usage.OutputTokens
						foundUsage = true
					}
				}
			}
		}
	}

	if foundUsage {
		modelStr := "unknown"
		if info.Model != nil {
			modelStr = *info.Model
		}
		log.Printf("[Token提取] 使用策略1-SSE解析: input=%d, output=%d, model=%s",
			info.InputTokens, info.OutputTokens, modelStr)
		return &info, nil
	}

	// 策略2: 正则提取
	return extractTokensByRegex(body)
}

// extractTokensByRegex 策略2: 使用正则表达式提取 token 信息
func extractTokensByRegex(body []byte) (*ClaudeUsageInfo, error) {
	var info ClaudeUsageInfo
	var foundUsage bool

	// 提取 model
	modelRegex := regexp.MustCompile(`"model"\s*:\s*"(claude-[^"]+)"`)
	if matches := modelRegex.FindSubmatch(body); len(matches) > 1 {
		modelStr := string(matches[1])
		info.Model = &modelStr
	}

	// 提取 input_tokens
	inputRegex := regexp.MustCompile(`"input_tokens"\s*:\s*(\d+)`)
	if matches := inputRegex.FindAllSubmatch(body, -1); len(matches) > 0 {
		// 使用最后一个匹配（最终统计）
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) >= 2 {
			if val, err := strconv.ParseUint(string(lastMatch[1]), 10, 64); err == nil {
				info.InputTokens = val
				foundUsage = true
			}
		}
	}

	// 提取 output_tokens
	outputRegex := regexp.MustCompile(`"output_tokens"\s*:\s*(\d+)`)
	if matches := outputRegex.FindAllSubmatch(body, -1); len(matches) > 0 {
		// 使用最后一个匹配（最终统计）
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) >= 2 {
			if val, err := strconv.ParseUint(string(lastMatch[1]), 10, 64); err == nil {
				info.OutputTokens = val
				foundUsage = true
			}
		}
	}

	if foundUsage {
		log.Printf("[Token提取] 使用策略2-正则提取: input=%d, output=%d", info.InputTokens, info.OutputTokens)
		return &info, nil
	}

	// 策略3: 根据内容长度估算
	return estimateTokensByContent(body, info.Model)
}

// estimateTokensByContent 策略3: 根据内容长度估算 token 数量
// 估算规则: 英文约 4 字符 = 1 token，中文约 2 字符 = 1 token
// 默认按 70% 输入 / 30% 输出比例分配
func estimateTokensByContent(body []byte, model *string) (*ClaudeUsageInfo, error) {
	contentLen := len(body)
	if contentLen == 0 {
		return nil, nil
	}

	// 粗略估算总 token 数（按平均 3 字符 = 1 token）
	estimatedTotal := uint64(contentLen / 3)

	// 按 70% 输入 / 30% 输出分配
	inputTokens := uint64(float64(estimatedTotal) * 0.7)
	outputTokens := uint64(float64(estimatedTotal) * 0.3)

	info := &ClaudeUsageInfo{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
	}

	log.Printf("[Token提取] 使用策略3-内容估算: input=%d, output=%d (总长度=%d字节)",
		inputTokens, outputTokens, contentLen)

	return info, nil
}
