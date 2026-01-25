package utils

import (
	"log"
	"regexp"
	"strings"
)

// isPasswordLabel 检测是否是密码标签而不是实际密码
func isPasswordLabel(password string) bool {
	lower := strings.ToLower(strings.TrimSpace(password))
	// 常见的密码标签
	labels := []string{"密码", "password", "pwd", "密码：", "password:", "pwd:", "密:", "密：", "登录密码", "登录密码："}
	for _, label := range labels {
		if lower == label {
			return true
		}
	}
	return false
}

// ExtractAccountsMultiLevel 从文本中多级提取账号信息
// 支持多种格式的账号/密码和 token 提取
// 返回 []map[string]string，每个 map 包含:
// - type: "account" 或 "token"
// - level: "high" (结构化匹配) 或 "low" (宽松匹配)
// - 对于 account: account_email, account_password
// - 对于 token: token
func ExtractAccountsMultiLevel(text string, fields []string) []map[string]string {
	log.Println("Entering ExtractAccountsMultiLevel with text length:", len(text), "and fields:", fields, "text:", text)
	var results []map[string]string

	// 规范化文本：替换全角冒号，处理换行
	text = strings.ReplaceAll(text, "：", ":")
	lines := strings.Split(text, "\n")

	// 高优先级正则模式 (结构化，如 "登录账号：email----登录密码：password")
	highPatterns := []struct {
		re    *regexp.Regexp
		isAcc bool // true: account, false: token
	}{
		{regexp.MustCompile(`(?i)登录账号[:\s]*([\w\.-]+@[\w\.-]+)[-]{2,}登录密码[:\s]*([^\s]+)`), true},
		{regexp.MustCompile(`(?i)账号[:\s]*([\w\.-]+@[\w\.-]+)[\s]+密码[:\s]*([^\s]+)`), true},
		{regexp.MustCompile(`(?i)号[:\s]*([\w\.-]+@[\w\.-]+)[\s]+密[:\s]*([^\s]+)`), true},
		{regexp.MustCompile(`(?i)key[:\s]*([\w\.-]+@[\w\.-]+)[\s]+密码[:\s]*([^\s]+)`), true},
		{regexp.MustCompile(`(?i)邮箱[:\s]*([\w\.-]+@[\w\.-]+)[\s]+密码[:\s]*([^\s]+)`), true},
		{regexp.MustCompile(`([\w\.-]+@[\w\.-]+)[-]{2,}([^\s]+)`), true}, // 宽松 email----password
		{regexp.MustCompile(`(?i)token[:\s]*([^\s]+)`), false},
	}

	// 低优先级模式 (宽松匹配，整个行扫描)
	lowAccRe := regexp.MustCompile(`([^\s@]+@[^\s@]+\.[^\s@]+)[\s]*([^\s]+)`) // email + possible password

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matched := false

		// 先尝试高优先级匹配
		for _, pat := range highPatterns {
			matches := pat.re.FindStringSubmatch(line)
			if len(matches) > 1 {
				rec := make(map[string]string)
				rec["level"] = "high"

				if pat.isAcc {
					email := strings.TrimSpace(matches[1])
					password := strings.TrimSpace(matches[2])

					// 跳过如果 password 是标签
					if isPasswordLabel(password) {
						continue
					}

					// 验证 email 格式 (简单检查)
					if !strings.Contains(email, "@") {
						continue
					}

					rec["type"] = "account"
					rec["account_email"] = email
					rec["account_password"] = password
				} else {
					token := strings.TrimSpace(matches[1])
					rec["type"] = "token"
					rec["token"] = token
				}

				results = append(results, rec)
				matched = true
				break
			}
		}

		if matched {
			continue
		}

		// 尝试低优先级匹配 (仅针对 account)
		matches := lowAccRe.FindStringSubmatch(line)
		if len(matches) > 2 {
			email := strings.TrimSpace(matches[1])
			password := strings.TrimSpace(matches[2])

			// 跳过标签
			if isPasswordLabel(password) {
				continue
			}

			rec := make(map[string]string)
			rec["type"] = "account"
			rec["level"] = "low"
			rec["account_email"] = email
			rec["account_password"] = password
			results = append(results, rec)
		}
	}

	return results
}
