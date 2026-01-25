package utils

import (
	"fmt"
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

type AccountField map[string]string

func ExtractAccountsMultiLevel(text string, field []string) []AccountField {
	if len(field) == 0 {
		return nil
	}

	sepRe := regexp.MustCompile(`([\-#=_\*]{3,})\s*`)
	text = sepRe.ReplaceAllString(text, " ")

	var results []AccountField
	seen := make(map[string]bool)

	// 级别 1：严格模式（带前缀，支持跨行）
	results = addMatches(results, seen, text,
		`(?is)`+ // 跨行 + 不区分大小写
			`(?:邮箱[：:]?\s*|账号[：:]?\s*|Email[：:]?\s*|号:\s*|key:\s*|登录账号[：:]?\s*)?`+
			`([\w\.-]+@[\w\.-]+\.[\w]{2,})`+
			`(?:\s*[-—–]\s*|\s*密码[：:]?\s*|\s*密:\s*|\s*\s*|\s*登录密码：|\s{2,}|\n\s*)`+
			`([!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?A-Za-z0-9]{8,})`,
		field, "level1 - strict", "account")

	// 级别 2：宽松模式（强制执行）
	results = addMatches(results, seen, text,
		`(?is)([\w\.-]+@[\w\.-]+\.[\w]{2,})\s*([!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?A-Za-z0-9]{8,})`,
		field, "level2 - loose", "account")

	// 如果您希望保留仅邮箱提取，可在此处取消注释
	if len(results) == 0 {
		emailsOnly := extractUniqueEmails(text)
		for _, email := range emailsOnly {
			key := email + "|"
			if seen[key] {
				continue
			}
			seen[key] = true

			m := make(AccountField)
			if len(field) >= 1 {
				m[field[0]] = email
			}
			if len(field) >= 2 {
				m[field[1]] = ""
			}
			m["level"] = "level3 - email only"
			m["type"] = "account"
			results = append(results, m)
		}
	}

	// level4：token 提取 - 强制执行，不受邮箱结果数量影响
	tokenRe := regexp.MustCompile(`(?i)token[：:]?\s*([^\s]+)`)
	tokenMatches := tokenRe.FindAllStringSubmatch(text, -1)

	for _, match := range tokenMatches {
		if len(match) < 2 {
			continue
		}
		tokenVal := strings.TrimSpace(match[1])
		if tokenVal == "" {
			continue
		}

		// 使用独立的去重键
		key := "token|" + tokenVal
		if seen[key] {
			continue
		}
		seen[key] = true

		m := make(AccountField)
		m["token"] = tokenVal
		m["level"] = "level4 - token"
		m["type"] = "token"
		results = append(results, m)
	}

	return results
}

func addMatches(dst []AccountField, seen map[string]bool, text, pattern string, field []string, level, rowType string) []AccountField {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) < len(field)+1 {
			continue
		}

		m := make(AccountField)
		for i, fname := range field {
			val := strings.TrimSpace(match[i+1])
			m[fname] = val
		}
		m["level"] = level
		m["type"] = rowType

		key := m[field[0]]
		if len(field) >= 2 {
			key += "|" + m[field[1]]
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		dst = append(dst, m)
	}

	return dst
}

func extractUniqueEmails(text string) []string {
	// 第一步：宽松预扫描，找出所有可能的邮箱，统计 TLD 最大长度
	wideRe := regexp.MustCompile(`(?i)([\w\.-]+@[\w\.-]+\.[\w]{2,})`)
	found := wideRe.FindAllString(text, -1)

	seen := make(map[string]struct{})
	maxTLD := 2 // 兜底值（.com .cn 等常见长度）

	for _, email := range found {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}

		// 计算 TLD 长度（最后一个 . 后面的字符数）
		lastDot := strings.LastIndex(email, ".")
		if lastDot == -1 {
			continue
		}
		tld := email[lastDot+1:]
		l := len(tld)
		if l > maxTLD {
			maxTLD = l
		}

		// 同时去重加入 seen（可选，视需求保留）
		if _, ok := seen[email]; !ok {
			seen[email] = struct{}{}
		}
	}

	// 第二步：使用动态 TLD 长度构建更严格的正则
	pattern := fmt.Sprintf(`(?i)([\w\.-]+@[\w\.-]+\.[\w]{%d,})`, maxTLD)
	strictRe := regexp.MustCompile(pattern)

	// 第三步：用严格正则重新扫描，得到最终去重结果
	strictFound := strictRe.FindAllString(text, -1)

	var unique []string
	seen = make(map[string]struct{}) // 清空或复用之前的 seen

	for _, e := range strictFound {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if _, ok := seen[e]; !ok {
			seen[e] = struct{}{}
			unique = append(unique, e)
		}
	}

	return unique
}

// extractTokens 提取所有 token: 后面的值
func extractTokens(text string) []string {
	// 匹配 token: 或 token： 后面的非空格连续字符串
	re := regexp.MustCompile(`(?i)token[：:]?\s*([^\s]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	var tokens []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		token := strings.TrimSpace(match[1])
		if token == "" {
			continue
		}
		if !seen[token] {
			seen[token] = true
			tokens = append(tokens, token)
		}
	}

	return tokens
}
