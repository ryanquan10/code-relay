package controller

import (
	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type importRequest struct {
	Platform string `json:"platform" binding:"required"`
	Text     string `json:"text" binding:"required"`
}

type providerInfo struct {
	SourceType    string
	UpstreamURL   string
	UpstreamToken string
}

func PublicImportSources(c *gin.Context) {
	var req importRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	platform := strings.TrimSpace(strings.ToLower(req.Platform))
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform required"})
		return
	}

	// 目前仅支持 xianyu
	if platform != "xianyu" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported platform; only xianyu allowed"})
		return
	}

	items := extractCodexLike(req.Text)
	if len(items) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "no codex/claude credentials found"})
		return
	}

	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	createdSourceIDs := make([]int64, 0, len(items))
	addedMappings := 0

	if err := db.Transaction(func(tx *gorm.DB) error {
		productsByType := map[string][]entity.Product{}

		for _, it := range items {
			// 1) 创建 account_source（按 upstream_url + upstream_token 去重）
			normalizedURL := strings.TrimSpace(strings.TrimRight(it.UpstreamURL, "/"))
			normalizedToken := strings.TrimSpace(it.UpstreamToken)
			altURL := normalizedURL + "/"

			var source entity.AccountSource
			err := tx.Where("(upstream_url = ? OR upstream_url = ?) AND upstream_token = ?", normalizedURL, altURL, normalizedToken).
				First(&source).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if errors.Is(err, gorm.ErrRecordNotFound) {
				srcName := buildSourceName(providerInfo{SourceType: it.SourceType, UpstreamURL: normalizedURL, UpstreamToken: normalizedToken})
				cfg := entity.AccountSourceConfig{APIURL: strPtr(normalizedURL), APIKey: strPtr(normalizedToken), HandlerType: "manual"}
				cfgBytes, _ := json.Marshal(cfg)
				source = entity.AccountSource{
					SourceName:    srcName,
					UpstreamURL:   strPtr(normalizedURL),
					UpstreamToken: strPtr(normalizedToken),
					SourceType:    it.SourceType,
					Config:        cfgBytes,
					Priority:      1,
					AutoRental:    false,
					Status:        1,
				}
				if err := tx.Create(&source).Error; err != nil {
					return err
				}
				createdSourceIDs = append(createdSourceIDs, source.ID)
			} else {
				// 已存在相同 upstream_url+upstream_token：若类型不一致，直接报错避免脏数据
				if source.SourceType != "" && source.SourceType != it.SourceType {
					return fmt.Errorf("account_source already exists with same upstream_url+upstream_token but different source_type (existing=%s, input=%s)", source.SourceType, it.SourceType)
				}
			}

			// 2) 使用既有 product（按 account_type），不在导入时新增 product
			prods, ok := productsByType[it.SourceType]
			if !ok {
				if err := tx.Where("account_type = ? AND status = 1", it.SourceType).Find(&prods).Error; err != nil {
					return err
				}
				if len(prods) == 0 {
					return fmt.Errorf("no active product found for account_type=%s", it.SourceType)
				}
				productsByType[it.SourceType] = prods
			}

			// 3) 为该 account_type 匹配到的所有 product 建立映射 account_source_procut
			for _, prod := range prods {
				var count int64
				if err := tx.Model(&entity.AccountSourceProcut{}).
					Where("product_id = ? AND source_id = ?", prod.ID, source.ID).
					Count(&count).Error; err != nil {
					return err
				}
				if count == 0 {
					m := entity.AccountSourceProcut{ProductID: prod.ID, SourceID: source.ID, Weight: 1}
					if err := tx.Create(&m).Error; err != nil {
						return err
					}
					addedMappings++
				}
			}
		}
		return nil
	}); err != nil {
		log.Printf("[PublicImportSources] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "created_sources": len(createdSourceIDs), "source_ids": createdSourceIDs, "added_mappings": addedMappings})
}

// 提取 codex/claude + URL + Token（容错多种排版）
func extractCodexLike(text string) []providerInfo {
	var results []providerInfo

	lines := strings.Split(strings.TrimSpace(text), "\n")
	var cleaned []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l != "" {
			cleaned = append(cleaned, l)
		}
	}

	// 模式1：三连行（codex/claude, URL, Token）
	for i := 0; i+2 < len(cleaned); i++ {
		l1, l2, l3 := cleaned[i], cleaned[i+1], cleaned[i+2]
		if src := normalizeSource(l1); src != "" {
			if u := extractURL(l2); u != "" {
				if t := extractToken(l3); t != "" {
					results = append(results, providerInfo{SourceType: src, UpstreamURL: u, UpstreamToken: t})
					i += 2
				}
			}
		}
	}

	// 模式1b：同一行含 codex/claude 与 URL；下一行或下两行出现 Token
	for i := 0; i < len(cleaned); i++ {
		l := cleaned[i]
		if src := normalizeSource(l); src != "" {
			if u := extractURL(l); u != "" {
				for j := 1; j <= 2 && i+j < len(cleaned); j++ {
					if t := extractToken(cleaned[i+j]); t != "" {
						results = append(results, providerInfo{SourceType: src, UpstreamURL: u, UpstreamToken: t})
						i += j
						break
					}
				}
			}
		}
	}

	// 模式2：单行（codex/claude + URL + Token 同行）
	reCombined := regexp.MustCompile(`(?i)(codex|claude)\s*([a-z][a-z0-9+.-]*://[^\s"']+?)(?:\s+|\s*[，,:;]\s*)([A-Za-z0-9_\-]{20,}|(?:cr_)?[A-Fa-f0-9]{40,})`)
	if ms := reCombined.FindAllStringSubmatch(text, -1); len(ms) > 0 {
		for _, m := range ms {
			if len(m) == 4 {
				results = append(results, providerInfo{SourceType: normalizeSource(m[1]), UpstreamURL: m[2], UpstreamToken: m[3]})
			}
		}
	}

	// 模式3：config.toml + auth.json（base_url + OPENAI_API_KEY/API_KEY）
	reBaseURL := regexp.MustCompile(`base_url\s*=\s*["']?([^"'\s]+)["']?`)
	reKey := regexp.MustCompile(`\"(?:OPENAI_API_KEY|API_KEY)\"\s*:\s*[\"']?([^\"'\s]+)[\"']?`)
	baseURLs := reBaseURL.FindAllStringSubmatch(text, -1)
	keys := reKey.FindAllStringSubmatch(text, -1)
	if len(baseURLs) > 0 && len(keys) > 0 {
		for _, bu := range baseURLs {
			for _, k := range keys {
				if strings.Contains(strings.ToLower(bu[0]), "codex") || strings.Contains(strings.ToLower(k[0]), "codex") {
					results = append(results, providerInfo{SourceType: "codex", UpstreamURL: bu[1], UpstreamToken: k[1]})
					break
				}
			}
		}
	}

	// 去重（URL+Token）
	seen := make(map[string]struct{}, len(results))
	dedup := make([]providerInfo, 0, len(results))
	for _, r := range results {
		key := strings.ToLower(r.UpstreamURL) + "|" + r.UpstreamToken
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dedup = append(dedup, r)
	}
	return dedup
}

func normalizeSource(s string) string {
	s = strings.ToLower(strings.TrimSpace(strings.Trim(s, "\"'")))
	switch {
	case s == "codex" || strings.Contains(s, "codex"):
		return "codex"
	case s == "claude" || strings.Contains(s, "claude"):
		return "claude"
	default:
		return ""
	}
}

func extractURL(s string) string {
	re := regexp.MustCompile(`(https?://[^\s"'><]+)`)
	if m := re.FindString(s); m != "" {
		return m
	}
	reLoose := regexp.MustCompile(`https?://[a-zA-Z0-9_.-]+(?:/[a-zA-Z0-9_./?-]*)?`)
	return reLoose.FindString(s)
}

func extractToken(s string) string {
	re := regexp.MustCompile(`(?:cr_)?[A-Za-z0-9_\-]{40,}`)
	return re.FindString(s)
}

func buildSourceName(p providerInfo) string {
	u, err := url.Parse(p.UpstreamURL)
	if err != nil || u.Host == "" {
		return p.SourceType + "-source"
	}
	host := u.Hostname()
	parts := strings.Split(host, ".")
	base := parts[0]
	if len(parts) >= 2 {
		base = parts[len(parts)-2]
	}
	return base + "-" + p.SourceType
}

func displayProductName(platform, sourceType string) string {
	switch sourceType {
	case "codex":
		return platform + " Codex 账户"
	case "claude":
		return platform + " Claude 账户"
	default:
		return platform + "-" + sourceType
	}
}

func strPtr(s string) *string { return &s }
