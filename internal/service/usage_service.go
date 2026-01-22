package service

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/repository"
	"fmt"
	"log"
	"time"
)

// UsageService 处理使用量相关的业务逻辑
type UsageService struct {
	usageRepo   *repository.UsageRepository
	accountRepo *repository.AccountRepository
}

// NewUsageService 创建 UsageService 实例
func NewUsageService() *UsageService {
	return &UsageService{
		usageRepo:   repository.NewUsageRepository(),
		accountRepo: repository.NewAccountRepository(),
	}
}

// RecordTokenUsage 记录 token 使用量
// customerToken: 客户的 token（Account.Token）

// tokens: 使用的 token 数量
// consume: 消耗的余额金额
func (s *UsageService) RecordTokenUsage(customerToken string, tokens uint64, consume float64) error {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)

	if err != nil {
		return fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 获取当前小时
	now := time.Now()

	// 3. 创建 Usage 记录
	usage := &entity.Usage{
		AccountID:  account.ID,
		Consume:    consume,
		CreateTime: now,
		UpdateTime: now,
	}

	// 4. 保存到数据库
	if err := s.usageRepo.Create(usage); err != nil {
		return fmt.Errorf("failed to create usage record: %w", err)
	}

	// 5. 增加已使用余额 used_balance（如果 consume > 0）
	if consume > 0 {
		if err := s.accountRepo.IncrementUsedBalance(account.ID, consume); err != nil {
			log.Printf("[警告] 增加 used_balance 失败: account_id=%d, consume=%.4f, error=%v",
				account.ID, consume, err)
			// 注意：这里 used_balance 更新失败不应该阻止 usage 记录的创建
			// 可以根据业务需求决定是否回滚 usage 记录
		} else {
			newUsedBalance := account.UsedBalance + consume
			remaining := account.Balance - newUsedBalance
			log.Printf("[余额] 账户 %d 使用 %.4f, UsedBalance: %.4f -> %.4f, 剩余: %.4f",
				account.ID, consume, account.UsedBalance, newUsedBalance, remaining)
		}
	}

	log.Printf("[Usage] 记录成功: account_id=%d, tokens=%d, consume=%.4f",
		account.ID, tokens, consume)

	return nil
}

// GetTodayUsageByToken 获取指定 token 今天的使用量统计
func (s *UsageService) GetTodayUsageByToken(customerToken string) (float64, error) {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return 0, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return 0, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 查询今天的总消费
	totalConsume, err := s.usageRepo.GetTodayTotalConsumeByAccountId(account.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get today's usage: %w", err)
	}

	return totalConsume, nil
}

// GetTodayUsageListByToken 获取指定 token 今天的所有使用记录
func (s *UsageService) GetTodayUsageListByToken(customerToken string) ([]entity.Usage, error) {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 查询今天的使用记录
	usageList, err := s.usageRepo.GetTodayUsageByAccountId(account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's usage list: %w", err)
	}

	return usageList, nil
}

// GetUsageByTokenAndDateRange 获取指定 token 在日期范围内的使用量
func (s *UsageService) GetUsageByTokenAndDateRange(customerToken string, startTime, endTime time.Time) (float64, error) {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return 0, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return 0, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 查询指定日期范围的总消费
	totalConsume, err := s.usageRepo.GetTotalConsumeByAccountIdAndDateRange(account.ID, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("failed to get usage by date range: %w", err)
	}

	return totalConsume, nil
}

// GetUsageListByTokenAndDateRange 获取指定 token 在日期范围内的使用记录列表
func (s *UsageService) GetUsageListByTokenAndDateRange(customerToken string, startTime, endTime time.Time) ([]entity.Usage, error) {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 查询指定日期范围的使用记录
	usageList, err := s.usageRepo.GetUsageByAccountIdAndDateRange(account.ID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage list by date range: %w", err)
	}

	return usageList, nil
}

// GetUsageByDates 获取指定 token 在多个日期的使用量总和
func (s *UsageService) GetUsageByDates(customerToken string, dates []string) (float64, error) {
	if len(dates) == 0 {
		return 0, fmt.Errorf("dates cannot be empty")
	}

	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return 0, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return 0, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 遍历所有日期，累加消费
	var totalConsume float64
	for _, dateStr := range dates {
		// 解析日期字符串
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			log.Printf("[警告] 日期格式错误: %s, error=%v", dateStr, err)
			continue
		}

		// 获取当天的开始和结束时间
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		// 查询该天的消费
		consume, err := s.usageRepo.GetTotalConsumeByAccountIdAndDateRange(account.ID, startOfDay, endOfDay)
		if err != nil {
			log.Printf("[警告] 查询日期 %s 的使用量失败: %v", dateStr, err)
			continue
		}

		totalConsume += consume
	}

	return totalConsume, nil
}

// GetTodayAllUsage 获取今天所有账户的使用记录（管理员功能）
func (s *UsageService) GetTodayAllUsage() ([]entity.Usage, error) {
	usageList, err := s.usageRepo.GetTodayUsageList()
	if err != nil {
		return nil, fmt.Errorf("failed to get today's all usage: %w", err)
	}

	return usageList, nil
}

// GetDailyUsageByToken 获取指定 token 按天分组的使用统计
// customerToken: 客户的 token
// startTime: 开始日期
// endTime: 结束日期
// 返回每天的消费总额和记录数
func (s *UsageService) GetDailyUsageByToken(customerToken string, startTime, endTime time.Time) ([]entity.DailyUsage, error) {
	// 1. 根据 customerToken 查找 Account
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get account by token: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("account not found for token: %s", maskToken(customerToken))
	}

	// 2. 查询按天分组的使用统计
	dailyUsages, err := s.usageRepo.GetDailyUsageByAccountId(account.ID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily usage: %w", err)
	}

	return dailyUsages, nil
}

// GetDailyUsageByTokenForDays 获取指定 token 最近 N 天的使用统计
// customerToken: 客户的 token
// days: 最近多少天（例如 7 表示最近 7 天）
func (s *UsageService) GetDailyUsageByTokenForDays(customerToken string, days int) ([]entity.DailyUsage, error) {
	if days <= 0 {
		days = 7 // 默认最近 7 天
	}

	// 计算时间范围
	now := time.Now()
	endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	startTime := endTime.AddDate(0, 0, -days+1) // 包含今天
	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())

	return s.GetDailyUsageByToken(customerToken, startTime, endTime)
}

// GetAllDailyUsage 获取所有账户按天分组的使用统计（管理员功能）
// startTime: 开始日期
// endTime: 结束日期
func (s *UsageService) GetAllDailyUsage(startTime, endTime time.Time) ([]entity.DailyUsage, error) {
	dailyUsages, err := s.usageRepo.GetDailyUsageGroupByDate(startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get all daily usage: %w", err)
	}

	return dailyUsages, nil
}

// maskToken 隐藏 token 的中间部分（用于日志）
func maskToken(token string) string {
	if len(token) <= 20 {
		return "***"
	}
	return token[:10] + "..." + token[len(token)-6:]
}
