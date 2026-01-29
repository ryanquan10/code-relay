package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
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
func parseClaudeJSONResponse(body []byte) (*ClaudeUsageInfo, error) {
	var resp ClaudeAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	info := &ClaudeUsageInfo{
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}
	if resp.Model != "" {
		info.Model = &resp.Model
	}

	return info, nil
}

// parseClaudeSSEResponse 解析 SSE 流式响应
// 支持格式: event: xxx\ndata: {...}
func parseClaudeSSEResponse(body []byte) (*ClaudeUsageInfo, error) {
	var info ClaudeUsageInfo
	var foundUsage bool

	// 正则提取 model（从 message_start 事件）
	modelRegex := regexp.MustCompile(`"model"\s*:\s*"([^"]+)"`)
	if matches := modelRegex.FindSubmatch(body); len(matches) > 1 {
		modelStr := string(matches[1])
		info.Model = &modelStr
	}

	// 正则提取 usage（优先从 message_delta 事件，因为它包含最终统计）
	// 匹配: "usage":{"input_tokens":13,"output_tokens":10}
	usageRegex := regexp.MustCompile(`"usage"\s*:\s*\{\s*"input_tokens"\s*:\s*(\d+)\s*,\s*"output_tokens"\s*:\s*(\d+)\s*\}`)
	matches := usageRegex.FindAllSubmatch(body, -1)

	// 使用最后一个匹配（message_delta 的 usage）
	if len(matches) > 0 {
		lastMatch := matches[len(matches)-1]
		if len(lastMatch) >= 3 {
			var inTokens, outTokens uint64
			json.Unmarshal(lastMatch[1], &inTokens)
			json.Unmarshal(lastMatch[2], &outTokens)
			info.InputTokens = inTokens
			info.OutputTokens = outTokens
			foundUsage = true
		}
	}

	if !foundUsage {
		return nil, nil
	}

	return &info, nil
}
