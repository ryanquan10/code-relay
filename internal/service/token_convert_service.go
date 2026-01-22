package service

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/repository"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// TokenConvertService 负责将客户端 token 转换为上流服务器地址和上流 token
type TokenConvertService struct {
	accountRepo       *repository.AccountRepository
	accountSourceRepo *repository.AccountSourceRepository
}

// UpstreamConfig 包含上流服务器的配置信息
type UpstreamConfig struct {
	UpstreamURL   string // 上流服务器地址
	UpstreamToken string // 上流服务器的 API Key/Token
}

// NewTokenConvertService 创建新的 TokenConvertService 实例
func NewTokenConvertService() *TokenConvertService {
	return &TokenConvertService{
		accountRepo:       repository.NewAccountRepository(),
		accountSourceRepo: repository.NewAccountSourceRepository(),
	}
}

// ConvertToken 根据客户端 token 获取上流服务器配置
// 逻辑：
// 1. 使用 token 从 Account 表查找账号
// 2. 从 Account 获取 SourceID
// 3. 使用 SourceID 从 AccountSource 表查找上流配置（解析 Config JSON）
func (s *TokenConvertService) ConvertToken(customerToken string) (*UpstreamConfig, error) {
	if customerToken == "" {
		return nil, fmt.Errorf("customer token is empty")
	}

	// 1. 根据 token 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		log.Printf("[TokenConvert] 查询账号失败: %v", err)
		return nil, fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		log.Printf("[TokenConvert] 未找到 token 对应的账号: %s", maskToken(customerToken))
		return nil, fmt.Errorf("account not found for token")
	}

	log.Printf("[TokenConvert] 找到账号 ID=%d, ProductID=%d, SourceID=%d",
		account.ID, account.ProductID, account.SourceID)

	// 2. 根据 SourceID 查找 AccountSource (获取上流配置)
	accountSource, err := s.accountSourceRepo.GetByID(account.SourceID)
	if err != nil {
		log.Printf("[TokenConvert] 查询 AccountSource 失败: %v", err)
		return nil, fmt.Errorf("failed to find account_source: %w", err)
	}
	if accountSource == nil {
		log.Printf("[TokenConvert] 未找到 AccountSource (SourceID=%d)", account.SourceID)
		return nil, fmt.Errorf("account_source not found")
	}

	upstreamURL := ""
	upstreamToken := ""
	if accountSource.UpstreamURL != nil {
		upstreamURL = strings.TrimSpace(*accountSource.UpstreamURL)
	}
	if accountSource.UpstreamToken != nil {
		upstreamToken = strings.TrimSpace(*accountSource.UpstreamToken)
	}

	// 3. 解析 Config JSON 获取 APIURL 和 APIKey (作为兼容回退)
	if upstreamURL == "" || upstreamToken == "" {
		var config entity.AccountSourceConfig
		if len(accountSource.Config) > 0 {
			if err := json.Unmarshal(accountSource.Config, &config); err != nil {
				log.Printf("[TokenConvert] 解析 AccountSource Config 失败: %v", err)
				return nil, fmt.Errorf("failed to parse account_source config: %w", err)
			}
		}

		if upstreamURL == "" && config.APIURL != nil {
			upstreamURL = strings.TrimSpace(*config.APIURL)
		}
		if upstreamToken == "" && config.APIKey != nil {
			upstreamToken = strings.TrimSpace(*config.APIKey)
		}
	}

	// 检查 upstream_url 是否存在
	if upstreamURL == "" {
		log.Printf("[TokenConvert] AccountSource 的 UpstreamURL 为空 (ID=%d)", accountSource.ID)
		return nil, fmt.Errorf("upstream_url is empty for account_source")
	}

	// 检查 upstream_token 是否存在
	if upstreamToken == "" {
		log.Printf("[TokenConvert] AccountSource 的 UpstreamToken 为空 (ID=%d)", accountSource.ID)
		return nil, fmt.Errorf("upstream_token is empty for account_source")
	}

	upstreamConfig := &UpstreamConfig{
		UpstreamURL:   upstreamURL,
		UpstreamToken: upstreamToken,
	}

	log.Printf("[TokenConvert] 转换成功: CustomerToken=%s -> UpstreamURL=%s, UpstreamToken=%s",
		maskToken(customerToken), upstreamConfig.UpstreamURL, maskToken(upstreamConfig.UpstreamToken))

	return upstreamConfig, nil
}
