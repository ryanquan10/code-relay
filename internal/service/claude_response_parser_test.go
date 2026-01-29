package service

import (
	"testing"
)

func TestParseClaudeSSEResponse(t *testing.T) {
	// 模拟官方 SSE 响应
	sseResponse := `event: message_start
data: {"message":{"content":[],"id":"msg_20260129185730","model":"claude-sonnet-4-5-20250929","role":"assistant","stop_reason":null,"stop_sequence":null,"type":"message","usage":{"input_tokens":13,"output_tokens":1}},"type":"message_start"}

event: ping
data: {"type":"ping"}

event: content_block_delta
data: {"delta":{"text":"1\n\n2\n\n3\n\n4\n\n5","type":"text_delta"},"index":0,"type":"content_block_delta"}

event: message_delta
data: {"delta":{"stop_reason":"end_turn","stop_sequence":null},"type":"message_delta","usage":{"input_tokens":13,"output_tokens":10}}

event: message_stop
data: {"type":"message_stop"}
`

	usageInfo, err := ParseClaudeResponse([]byte(sseResponse), true)

	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if usageInfo == nil {
		t.Fatal("未找到 usage 信息")
	}

	// 验证 tokens
	if usageInfo.InputTokens != 13 {
		t.Errorf("InputTokens 错误: 期望 13, 实际 %d", usageInfo.InputTokens)
	}

	if usageInfo.OutputTokens != 10 {
		t.Errorf("OutputTokens 错误: 期望 10, 实际 %d", usageInfo.OutputTokens)
	}

	// 验证 model
	if usageInfo.Model == nil {
		t.Error("Model 为 nil")
	} else if *usageInfo.Model != "claude-sonnet-4-5-20250929" {
		t.Errorf("Model 错误: 期望 claude-sonnet-4-5-20250929, 实际 %s", *usageInfo.Model)
	}

	t.Logf("✅ 测试通过: InputTokens=%d, OutputTokens=%d, Model=%s",
		usageInfo.InputTokens, usageInfo.OutputTokens, *usageInfo.Model)
}
