package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
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
// SSE 格式: data: {...}\n\n
func parseClaudeSSEResponse(body []byte) (*ClaudeUsageInfo, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	var info ClaudeUsageInfo
	var foundUsage bool

	for scanner.Scan() {
		line := scanner.Text()

		// SSE 格式: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			continue
		}

		var event ClaudeSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue // 跳过无法解析的行
		}

		// 从 message_stop 或 message_delta 事件中提取 usage
		if event.Type == "message_stop" || event.Type == "message_delta" {
			if event.Usage != nil {
				info.InputTokens = event.Usage.InputTokens
				info.OutputTokens = event.Usage.OutputTokens
				foundUsage = true
			}
		}

		// 从 message_start 事件中提取 model 和初始 usage
		if event.Type == "message_start" && event.Message != nil {
			if event.Message.Model != "" {
				info.Model = &event.Message.Model
			}
			if event.Message.Usage.InputTokens > 0 {
				info.InputTokens = event.Message.Usage.InputTokens
				foundUsage = true
			}
		}
	}

	if !foundUsage {
		return nil, nil // 未找到官方 token 信息，返回 nil（将使用估算）
	}

	return &info, nil
}
