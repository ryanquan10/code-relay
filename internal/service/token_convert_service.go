package service

import (
	"codex-relay/internal/repository"
	"fmt"
	"log"
)

// TokenConvertService 负责将客户端 token 转换为上流服务器地址和上流 token
type TokenConvertService struct {
	accountRepo        *repository.AccountRepository
	accountSourceRepo  *repository.AccountSourceRepository
	productSourceRepo  *repository.ProductSourceRepository
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
		productSourceRepo: repository.NewProductSourceRepository(),
	}
}

// ConvertToken 根据客户端 token 获取上流服务器配置
// 逻辑：
// 1. 使用 token 从 Account 表查找账号
// 2. 从 Account 获取 ProductID 和 SourceID
// 3. 使用 ProductID 和 SourceID 从 ProductSource 表查找上流配置
// 4. 使用 SourceID 从 AccountSource 表查找上流 token (API Key)
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

	// 2. 根据 ProductID 和 SourceID 查找 ProductSource (获取上流 URL)
	productSource, err := s.productSourceRepo.GetByProductAndSource(account.ProductID, account.SourceID)
	if err != nil {
		log.Printf("[TokenConvert] 查询 ProductSource 失败: %v", err)
		return nil, fmt.Errorf("failed to find product_source: %w", err)
	}
	if productSource == nil {
		log.Printf("[TokenConvert] 未找到 ProductSource (ProductID=%d, SourceID=%d)",
			account.ProductID, account.SourceID)
		return nil, fmt.Errorf("product_source not found")
	}

	// 检查 upstream_url 是否存在
	if productSource.UpstreamURL == nil || *productSource.UpstreamURL == "" {
		log.Printf("[TokenConvert] ProductSource 的 UpstreamURL 为空 (ID=%d)", productSource.ID)
		return nil, fmt.Errorf("upstream_url is empty in product_source")
	}

	// 3. 根据 SourceID 查找 AccountSource (获取上流 API Key)
	accountSource, err := s.accountSourceRepo.GetByID(account.SourceID)
	if err != nil {
		log.Printf("[TokenConvert] 查询 AccountSource 失败: %v", err)
		return nil, fmt.Errorf("failed to find account_source: %w", err)
	}
	if accountSource == nil {
		log.Printf("[TokenConvert] 未找到 AccountSource (SourceID=%d)", account.SourceID)
		return nil, fmt.Errorf("account_source not found")
	}

	// 检查 api_key 是否存在
	if accountSource.APIKey == nil || *accountSource.APIKey == "" {
		log.Printf("[TokenConvert] AccountSource 的 APIKey 为空 (ID=%d)", accountSource.ID)
		return nil, fmt.Errorf("api_key is empty in account_source")
	}

	upstreamConfig := &UpstreamConfig{
		UpstreamURL:   *productSource.UpstreamURL,
		UpstreamToken: *accountSource.APIKey,
	}

	log.Printf("[TokenConvert] 转换成功: CustomerToken=%s -> UpstreamURL=%s, UpstreamToken=%s",
		maskToken(customerToken), upstreamConfig.UpstreamURL, maskToken(upstreamConfig.UpstreamToken))

	return upstreamConfig, nil
}
