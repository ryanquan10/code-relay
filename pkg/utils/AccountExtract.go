package utils

import (
	"regexp"
	"strings"
)

// AccountField 用 map 表示一条账号信息
type AccountField map[string]string

// extractAccountsMultiLevel 返回多条提取结果，按 field 顺序映射捕获组
func extractAccountsMultiLevel(text string, field []string) []AccountField {
	if len(field) == 0 {
		return nil
	}

	var results []AccountField
	seen := make(map[string]bool) // key = "email|password" （或更多字段）

	// ── 级别 1：最严格（带前缀标签，如“邮箱：”“密码：”）
\tresults = addMatches(results, seen, text,
		`(?i)(?:邮箱[：:]?\s*|账号[：:]?\s*|Email[：:]?\s*|号:\s*|key:\s*)?`+
			`([\w\.-]+@[\w\.-]+\.[\w]{2,})`+
			`(?:\s*[-—–]\s*|\s*密码[：:]?\s*|\s*密:\s*|\s*----\s*|\s*----登录密码：)\s*`+
			`([!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?A-Za-z0-9]{8,})`,
		field, "level1 - strict")

	// ── 级别 2：宽松（邮箱 + 密码直接挨着或间隔少量字符）
	if len(results) == 0 { // 可选：如果已有严格匹配，可注释掉此条件以强制收集更多
	\tresults = addMatches(results, seen, text,
			`(?i)([\w\.-]+@[\w\.-]+\.[\w]{2,})\s*([!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?A-Za-z0-9]{8,})`,
			field, "level2 - loose")
	}

	// ── 级别 3：仅邮箱（无密码）
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
		results = append(results, m)
	}

	return results
}

// addMatches 统一提取并加入结果（支持多条）
func addMatches(dst []AccountField, seen map[string]bool, text, pattern string, field []string, level string) []AccountField {
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

		// 构建去重 key（可根据需要包含更多字段）
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

// extractUniqueEmails 提取所有不重复的邮箱
func extractUniqueEmails(text string) []string {
	re := regexp.MustCompile(`(?i)([\w\.-]+@[\w\.-]+\.[\w]{2,})`)
	found := re.FindAllString(text, -1)

	seen := make(map[string]struct{})
	var unique []string
	for _, e := range found {
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

// ExtractAccounts is the exported entry to parse text by given field order.
func ExtractAccounts(text string, field []string) []AccountField {
    return extractAccountsMultiLevel(text, field)
}