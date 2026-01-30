package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ProviderInfo struct {
	SourceType    string `json:"source_type"`
	UpstreamURL   string `json:"upstream_url"`
	UpstreamToken string `json:"upstream_token"`
}

func extractCodexInfo(text string) []ProviderInfo {
	var results []ProviderInfo

	// 清理多余空白，方便后续匹配
	lines := strings.Split(strings.TrimSpace(text), "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	// 模式1：连续三行（最常见格式）
	// codex
	// https://xxx
	// cr_xxxx...
	for i := 0; i+2 < len(cleaned); i++ {
		l1 := cleaned[i]
		l2 := cleaned[i+1]
		l3 := cleaned[i+2]

		source := normalizeSource(l1)
		if source == "" {
			continue
		}

		url := extractURL(l2)
		if url == "" {
			continue
		}

		token := extractToken(l3)
		if token == "" {
			continue
		}

		results = append(results, ProviderInfo{
			SourceType:    source,
			UpstreamURL:   url,
			UpstreamToken: token,
		})
		// 跳过已使用的两行
		i += 2
	}

	// 模式2：单行中包含 codex/claude + URL + token（较少见）
	reCombined := regexp.MustCompile(`(?i)(codex|claude)\s*([a-z][a-z0-9+.-]*://[^\s"']+?)(?:\s+|\s*[，,:;]\s*)([a-z0-9]{40,})`)
	matches := reCombined.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) == 4 {
			results = append(results, ProviderInfo{
				SourceType:    normalizeSource(m[1]),
				UpstreamURL:   m[2],
				UpstreamToken: m[3],
			})
		}
	}

	// 模式3：config.toml 或 auth.json 中的 base_url + key（备用）
	reBaseURL := regexp.MustCompile(`base_url\s*=\s*["']?([^"'\s]+)["']?`)
	reKey := regexp.MustCompile(`"(?:OPENAI_API_KEY|API_KEY)"\s*:\s*["']?([^"'\s]+)["']?`)

	baseURLs := reBaseURL.FindAllStringSubmatch(text, -1)
	keys := reKey.FindAllStringSubmatch(text, -1)

	if len(baseURLs) > 0 && len(keys) > 0 {
		for _, bu := range baseURLs {
			for _, k := range keys {
				if strings.Contains(bu[0], "codex") || strings.Contains(k[0], "codex") {
					results = append(results, ProviderInfo{
						SourceType:    "codex",
						UpstreamURL:   bu[1],
						UpstreamToken: k[1],
					})
					break
				}
			}
		}
	}

	return results
}

func normalizeSource(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, `"'`)
	if s == "codex" || s == "claude" {
		return s
	}
	if strings.Contains(s, "codex") {
		return "codex"
	}
	if strings.Contains(s, "claude") {
		return "claude"
	}
	return ""
}

func extractURL(s string) string {
	re := regexp.MustCompile(`(https?://[^\s"'><]+)`)
	m := re.FindString(s)
	if m != "" {
		return m
	}
	// 宽松匹配
	reLoose := regexp.MustCompile(`https?://[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_./?-]*)?`)
	mLoose := reLoose.FindString(s)
	return mLoose
}

func extractToken(s string) string {
	// 匹配 cr_ 开头或纯 40+ 位 hex
	re := regexp.MustCompile(`(?:cr_)?[a-f0-9]{40,}`)
	m := re.FindString(s)
	return m
}

func main() {
	// 替换为您的实际文本
	input := `
codex接口地址（URL）：https://crss.nanashiwang.com/openai/
卡券密钥：卡号：日卡1密码：cr_e0a2b0ec9d2d96f076957e98aa6f99871274d08f4c15aa4f1e27821a59e75481

把
codex
https://crss.nanashiwang.com/openai/
cr_e0a2b0ec9d2d96f076957e98aa6f99871274d08f4c15aa4f1e27821a59e75481

C:\Users\用户名\.codex\config.toml
model_provider = "fox"
base_url = "https://keyswift.top/codex/v1"

C:\Users\用户名\.codex\auth.json
{
"OPENAI_API_KEY": "codex-1d-83e49a3ffea26496c0e7e15a9023790e"
}
    `

	results := extractCodexInfo(input)

	if len(results) == 0 {
		fmt.Println("未匹配到有效的 codex/claude 配置")
		return
	}

	for i, r := range results {
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Printf("匹配结果 %d:\n%s\n\n", i+1, string(b))
	}
}
