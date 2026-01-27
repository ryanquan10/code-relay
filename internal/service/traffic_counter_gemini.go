package service

import (
	"io"
)

// GeminiTrafficCounter 是对通用 TrafficCounter 的类型别名，保持接口一致
// 这样如果未来 Gemini 需要不同的计价或行为，可在此定制而不影响 Codex

type GeminiTrafficCounter = TrafficCounter

func NewGeminiTrafficCounter(customerToken string) *GeminiTrafficCounter {
	return NewTrafficCounter(customerToken)
}

func NewGeminiCountingReadCloser(r io.ReadCloser, counter *GeminiTrafficCounter, dir string) io.ReadCloser {
	return NewCountingReadCloser(r, (*TrafficCounter)(counter), dir)
}
