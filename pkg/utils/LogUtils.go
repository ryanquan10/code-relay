package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

// LogRequestDetails 打印 HTTP 请求的完整调试信息，包括方法、URL、Header、Query 和 Body。
// 该函数会读取 Body 并在结束后自动复位 r.Body，以便后续 handler 继续使用。
func LogRequestDetails(r *http.Request) {
	// 1. 请求基本信息
	log.Printf("[delivery][debug] %s %s", r.Method, r.URL.String())

	// 2. 打印所有 Header（逐条）
	log.Printf("[delivery][debug] Headers:")
	for key, values := range r.Header {
		log.Printf("[delivery][debug]   %s: %s", key, strings.Join(values, ", "))
	}

	// 3. 打印 Query 参数（优先使用解析后的结构化形式）
	query := r.URL.Query()
	if len(query) > 0 {
		log.Printf("[delivery][debug] Query parameters:")
		for key, values := range query {
			log.Printf("[delivery][debug]   %s: %s", key, strings.Join(values, ", "))
		}
	} else if r.URL.RawQuery != "" {
		// 如果解析失败但原始 query 字符串存在，则打印原始内容
		log.Printf("[delivery][debug] Raw query (unparsed): %s", r.URL.RawQuery)
	}

	// 4. 读取并打印 Body，同时确保复位
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[delivery][debug] Failed to read request body: %v", err)
		// 即使读取失败，也尝试恢复一个空 Body
		r.Body = io.NopCloser(bytes.NewBuffer(nil))
		return
	}

	// 必须在 defer 中复位 Body
	defer func() {
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}()

	// 仅当 Body 非空时打印内容
	if len(bodyBytes) > 0 {
		bodyStr := string(bodyBytes)

		// 尝试判断是否为 JSON，若是则格式化输出（可选提升可读性）
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "  ", "  "); err == nil && prettyJSON.Len() > 0 {
			log.Printf("[delivery][debug] Request body (formatted JSON):\n%s", prettyJSON.String())
		} else {
			// 非 JSON 或格式化失败，直接打印原始字符串
			log.Printf("[delivery][debug] Request body (raw):\n%s", bodyStr)
		}
	} else {
		log.Printf("[delivery][debug] Request body: <empty>")
	}
}
