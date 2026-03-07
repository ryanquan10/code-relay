package service

import (
	"codex-relay/internal/redis"
	"codex-relay/internal/repository"
	"codex-relay/pkg/entity"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// TokenConvertService 负责将客户端 token 转换为上流服务器地址和上流 token
type TokenConvertService struct {
	accountRepo          *repository.AccountRepository
	accountSourceRepo    *repository.AccountSourceRepository
	sourceProductRepo    *repository.AccountSourceProductRepository
	errorLogRepo         *repository.UpstreamErrorLogRepository
	redisPriorityTTL     time.Duration // Redis 优先级降级的过期时间（默认 10 分钟）
	enablePrioritySwitch bool          // 是否启用优先级切换
}

// UpstreamConfig 包含上流服务器的配置信息
type UpstreamConfig struct {
	UpstreamURL   string // 上流服务器地址
	UpstreamToken string // 上流服务器的 API Key/Token
	AccountID     uint64 // 账户 ID（用于错误日志）
	SourceID      int64  // 上游源 ID
	SourceType    string // 上游源类型
	ProductID     int64  // 产品 ID
}

// NewTokenConvertService 创建新的 TokenConvertService 实例
func NewTokenConvertService() *TokenConvertService {
	return &TokenConvertService{
		accountRepo:          repository.NewAccountRepository(),
		accountSourceRepo:    repository.NewAccountSourceRepository(),
		sourceProductRepo:    repository.NewAccountSourceProductRepository(),
		errorLogRepo:         repository.NewUpstreamErrorLogRepository(),
		redisPriorityTTL:     10 * time.Minute, // 默认 10 分钟
		enablePrioritySwitch: true,             // 默认启用优先级切换
	}
}

// ConvertTokenAndCheck 根据客户端 token 获取上流服务器配置
// 逻辑：
// 1. 使用 token 从 Account 表查找账号
// 2. 从 Account 获取 SourceID 和 ProductID
// 3. 根据优先级降级机制选择可用的上游配置
func (s *TokenConvertService) ConvertTokenAndCheck(customerToken string, requestPath string) (*UpstreamConfig, error) {
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

	log.Printf("[TokenConvert] 找到账号 ID=%d, ProductID=%d, SourceID=%d", account.ID, account.ProductID, account.SourceID)

	// 额度检查：UsedBalance > Balance 视为余额不足
	if account.UsedBalance > account.Balance {
		log.Printf("[TokenConvert] 账号余额不足 [ID: %d, Balance: %.2f, UsedBalance: %.2f, 欠费: %.2f]",
			account.ID, account.Balance, account.UsedBalance, account.UsedBalance-account.Balance)
		return nil, fmt.Errorf("insufficient balance")
	}

	// 过期检查：命中任一过期条件即拒绝使用
	if s.isAccountExpired(account, time.Now()) {
		return nil, fmt.Errorf("account has expired")
	}

	// 2. 根据 ProductID 查询所有可用的上游源（按优先级排序）
	sources, err := s.sourceProductRepo.GetSourcesByProductID(account.ProductID)
	if err != nil {
		log.Printf("[TokenConvert] 查询 Product 的上游源失败: %v", err)
		return nil, fmt.Errorf("failed to find sources for product: %w", err)
	}

	if len(sources) == 0 {
		log.Printf("[TokenConvert] Product (ID=%d) 没有配置任何上游源", account.ProductID)
		// 降级：使用原始的 SourceID 查询（兼容旧逻辑）
		return s.getUpstreamConfigBySourceID(account, requestPath)
	}

	// 3. 优先级降级逻辑：根据 Redis 选择可用的上游
	selectedSource := s.selectAvailableSource(sources)
	if selectedSource == nil {
		log.Printf("[TokenConvert] 所有上游源都不可用，降级使用默认配置")
		return s.getUpstreamConfigBySourceID(account, requestPath)
	}

	// 限流检查：检查是否超过 token 使用限制
	if exceeded, err := IsTokenLimitExceeded(customerToken, selectedSource.SourceType); err == nil && exceeded {
		log.Printf("[TokenConvert] 账号已超过限流 [Token: %s]", maskToken(customerToken))
		return nil, fmt.Errorf("rate limit exceeded")
	}

	// 4. 构建 UpstreamConfig
	upstreamConfig, err := s.buildUpstreamConfig(account, selectedSource, requestPath)
	if err != nil {
		log.Printf("[TokenConvert] 构建上游配置失败: %v", err)
		return nil, err
	}

	log.Printf("[TokenConvert] 转换成功: CustomerToken=%s -> SourceID=%d, UpstreamURL=%s",
		maskToken(customerToken), selectedSource.ID, upstreamConfig.UpstreamURL)

	return upstreamConfig, nil
}

func (s *TokenConvertService) isAccountExpired(account *entity.Account, now time.Time) bool {
	if account == nil {
		return true
	}

	// 规则1: 显式过期时间到达即过期（优先级最高）
	if account.ExpireDate != nil {
		expireAt := *account.ExpireDate
		// 到达过期时刻即视为过期
		if !now.Before(expireAt) {
			log.Printf("[TokenConvert] 账号已过期(ExpireDate) [ID: %d, Now: %s, ExpireDate: %s]",
				account.ID, now.Format("2006-01-02 15:04:05"), expireAt.Format("2006-01-02 15:04:05"))
			return true
		}
	}

	// 规则2: 基于 start_time + expire_days 的相对过期
	if account.StartTime != nil {
		if account.ExpireDays <= 0 {
			log.Printf("[TokenConvert] 账号已过期(ExpireDays<=0) [ID: %d, StartTime: %s, ExpireDays: %d]",
				account.ID, account.StartTime.Format("2006-01-02 15:04:05"), account.ExpireDays)
			return true
		}
		expireAt := account.StartTime.Add(time.Duration(account.ExpireDays) * 24 * time.Hour)
		if !now.Before(expireAt) {
			log.Printf("[TokenConvert] 账号已过期(StartTime+ExpireDays) [ID: %d, StartTime: %s, ExpireDays: %d, ExpireAt: %s, Now: %s]",
				account.ID,
				account.StartTime.Format("2006-01-02 15:04:05"),
				account.ExpireDays,
				expireAt.Format("2006-01-02 15:04:05"),
				now.Format("2006-01-02 15:04:05"))
			return true
		}
	}

	return false
}

// getUpstreamConfigBySourceID 根据 account.SourceID 查询上游配置（兼容旧逻辑）
func (s *TokenConvertService) getUpstreamConfigBySourceID(account *entity.Account, requestPath string) (*UpstreamConfig, error) {
	accountSource, err := s.accountSourceRepo.GetByID(account.SourceID)
	if err != nil {
		log.Printf("[TokenConvert] 查询 AccountSource 失败: %v", err)
		return nil, fmt.Errorf("failed to find account_source: %w", err)
	}
	if accountSource == nil {
		log.Printf("[TokenConvert] 未找到 AccountSource (SourceID=%d)", account.SourceID)
		return nil, fmt.Errorf("account_source not found")
	}

	return s.buildUpstreamConfig(account, accountSource, requestPath)
}

// buildUpstreamConfig 从 AccountSource 构建 UpstreamConfig
func (s *TokenConvertService) buildUpstreamConfig(account *entity.Account, source *entity.AccountSource, requestPath string) (*UpstreamConfig, error) {
	upstreamURL := ""
	upstreamToken := ""
	if source.UpstreamURL != nil {
		upstreamURL = strings.TrimSpace(*source.UpstreamURL)
	}
	if source.UpstreamToken != nil {
		upstreamToken = strings.TrimSpace(*source.UpstreamToken)
	}

	// 解析 Config JSON 获取 APIURL 和 APIKey (作为兼容回退)
	if upstreamURL == "" || upstreamToken == "" {
		var config entity.AccountSourceConfig
		if len(source.Config) > 0 {
			if err := json.Unmarshal(source.Config, &config); err != nil {
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
		log.Printf("[TokenConvert] AccountSource 的 UpstreamURL 为空 (ID=%d)", source.ID)
		return nil, fmt.Errorf("upstream_url is empty for account_source")
	}

	// 检查 upstream_token 是否存在
	if upstreamToken == "" {
		log.Printf("[TokenConvert] AccountSource 的 UpstreamToken 为空 (ID=%d)", source.ID)
		return nil, fmt.Errorf("upstream_token is empty for account_source")
	}

	// 规范化 URL：移除末尾的斜杠
	upstreamURL = strings.TrimRight(upstreamURL, "/")

	// 拼接请求路径
	// 如果请求路径以 /codex 开头，先剥离 /codex 前缀
	if strings.HasPrefix(requestPath, "/codex/") {
		requestPath = strings.TrimPrefix(requestPath, "/codex")
	} else if requestPath == "/codex" {
		requestPath = ""
	}

	// 如果 upstreamURL 已经包含了 /codex/v1 或 /api/v1 等路径
	// 而 requestPath 也是 /v1/... 开头，则直接替换
	// 例如：upstreamURL = https://openrouter.ai/api/v1, requestPath = /v1/chat/completions
	// 应该变成：https://openrouter.ai/api/v1/chat/completions
	if requestPath != "" && strings.HasPrefix(requestPath, "/v1/") {
		// 检查 upstreamURL 是否以 /v1 结尾
		if strings.HasSuffix(upstreamURL, "/v1") || strings.HasSuffix(upstreamURL, "/api/v1") {
			// 去掉 requestPath 开头的 /v1
			requestPath = strings.TrimPrefix(requestPath, "/v1")
		}
	}

	// 确保请求路径以 / 开头
	if requestPath != "" && !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	// 拼接完整的上游 URL
	fullUpstreamURL := upstreamURL + requestPath

	upstreamConfig := &UpstreamConfig{
		UpstreamURL:   fullUpstreamURL,
		UpstreamToken: upstreamToken,
		AccountID:     account.ID,
		SourceID:      source.ID,
		SourceType:    source.SourceType,
		ProductID:     account.ProductID,
	}

	return upstreamConfig, nil
}

// selectAvailableSource 根据优先级降级机制选择可用的上游源
func (s *TokenConvertService) selectAvailableSource(sources []entity.AccountSource) *entity.AccountSource {
	if !s.enablePrioritySwitch {
		// 未启用优先级切换，直接返回第一个（默认优先级最高）
		if len(sources) > 0 {
			return &sources[0]
		}
		return nil
	}

	// 从 Redis 获取每个 source 的降级优先级
	for i := range sources {
		source := &sources[i]
		degradedPriority := s.getDegradedPriority(source.ID)

		// 如果该 source 没有被降级（degradedPriority == 0），则使用它
		if degradedPriority == 0 {
			log.Printf("[TokenConvert] 选择上游源: ID=%d, Priority=%d (未降级)", source.ID, source.Priority)
			return source
		}

		log.Printf("[TokenConvert] 上游源 ID=%d 已降级，跳过 (DegradedPriority=%d)", source.ID, degradedPriority)
	}

	// 所有源都被降级，返回第一个（容错）
	if len(sources) > 0 {
		log.Printf("[TokenConvert] 所有上游源都已降级，使用第一个作为降级方案: ID=%d", sources[0].ID)
		return &sources[0]
	}

	return nil
}

// getDegradedPriority 从 Redis 获取上游源的降级优先级
// 返回值：0 表示未降级，> 0 表示已降级
func (s *TokenConvertService) getDegradedPriority(sourceID int64) int {
	rdb := redis.Client()
	if rdb == nil {
		// Redis 不可用，降级逻辑失效
		return 0
	}

	ctx := redis.Context()
	key := fmt.Sprintf("upstream:priority:%d", sourceID)

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		// key 不存在或其他错误，视为未降级
		return 0
	}

	priority, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("[TokenConvert] Redis 优先级值非法 (key=%s, value=%s), 视为未降级", key, val)
		return 0
	}

	return priority
}

// DegradeSource 降级指定的上游源（设置 Redis 过期时间）
func (s *TokenConvertService) DegradeSource(sourceID int64) error {
	rdb := redis.Client()
	if rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}

	ctx := redis.Context()
	key := fmt.Sprintf("upstream:priority:%d", sourceID)

	// 设置降级标记，值为 1 表示已降级
	degradedPriority := 1
	err := rdb.Set(ctx, key, degradedPriority, s.redisPriorityTTL).Err()
	if err != nil {
		log.Printf("[TokenConvert] 降级上游源失败 (SourceID=%d): %v", sourceID, err)
		return fmt.Errorf("failed to degrade source %d: %w", sourceID, err)
	}

	log.Printf("[TokenConvert] 上游源已降级 (SourceID=%d, TTL=%v)", sourceID, s.redisPriorityTTL)
	return nil
}

// LogUpstreamError 记录上游错误日志
func (s *TokenConvertService) LogUpstreamError(config *UpstreamConfig, requestPath string, statusCode int, errorMessage string) error {
	if config == nil {
		return fmt.Errorf("upstream config is nil")
	}

	now := time.Now()
	errorLog := &entity.UpstreamErrorLog{
		AccountID:    int64(config.AccountID),
		SourceID:     config.SourceID,
		SourceType:   config.SourceType,
		UpstreamURL:  &config.UpstreamURL,
		RequestPath:  &requestPath,
		StatusCode:   &statusCode,
		ErrorMessage: &errorMessage,
		RequestTime:  &now,
	}

	if err := s.errorLogRepo.Create(errorLog); err != nil {
		log.Printf("[TokenConvert] 记录上游错误日志失败: %v", err)
		return err
	}

	log.Printf("[TokenConvert] 已记录上游错误: SourceID=%d, StatusCode=%d, Path=%s", config.SourceID, statusCode, requestPath)
	return nil
}

// HandleUpstreamError 处理上游错误（记录日志 + 降级）
func (s *TokenConvertService) HandleUpstreamError(config *UpstreamConfig, requestPath string, statusCode int, errorMessage string) {
	// 记录错误日志
	if err := s.LogUpstreamError(config, requestPath, statusCode, errorMessage); err != nil {
		log.Printf("[TokenConvert] 记录上游错误失败: %v", err)
	}

	// 降级上游源
	if s.enablePrioritySwitch {
		if err := s.DegradeSource(config.SourceID); err != nil {
			log.Printf("[TokenConvert] 降级上游源失败: %v", err)
		}
	}
}
