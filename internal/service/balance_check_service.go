package service

import (
	"codex-relay/internal/repository"
	"fmt"
	"log"
)

// BalanceCheckService 余额检查服务
type BalanceCheckService struct {
	accountRepo *repository.AccountRepository
}

// NewBalanceCheckService 创建余额检查服务实例
func NewBalanceCheckService() *BalanceCheckService {
	return &BalanceCheckService{
		accountRepo: repository.NewAccountRepository(),
	}
}

// CheckBalance 检查账户余额是否充足
// 返回 true 表示余额充足，false 表示余额不足
func (s *BalanceCheckService) CheckBalance(customerToken string) (bool, error) {
	if customerToken == "" {
		return false, fmt.Errorf("customer token is empty")
	}

	// 根据 token 查找账户
	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		log.Printf("[BalanceCheck] 查询账号失败: %v", err)
		return false, fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		log.Printf("[BalanceCheck] 未找到 token 对应的账号")
		return false, fmt.Errorf("account not found for token")
	}

	// 检查余额：used_balance > balance 表示余额不足
	if account.UsedBalance > account.Balance {
		log.Printf("[BalanceCheck] 账号余额不足 [ID: %d, Balance: %.2f, UsedBalance: %.2f, 欠费: %.2f]",
			account.ID, account.Balance, account.UsedBalance, account.UsedBalance-account.Balance)
		return false, nil
	}

	// 余额充足
	remainingBalance := account.Balance - account.UsedBalance
	log.Printf("[BalanceCheck] 账号余额充足 [ID: %d, Balance: %.2f, UsedBalance: %.2f, Remaining: %.2f]",
		account.ID, account.Balance, account.UsedBalance, remainingBalance)
	return true, nil
}

// GetBalanceInfo 获取账户余额信息（用于日志和调试）
func (s *BalanceCheckService) GetBalanceInfo(customerToken string) (balance, usedBalance, remaining float64, err error) {
	if customerToken == "" {
		return 0, 0, 0, fmt.Errorf("customer token is empty")
	}

	account, err := s.accountRepo.GetByToken(customerToken)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to find account: %w", err)
	}
	if account == nil {
		return 0, 0, 0, fmt.Errorf("account not found for token")
	}

	balance = account.Balance
	usedBalance = account.UsedBalance
	remaining = balance - usedBalance
	return balance, usedBalance, remaining, nil
}
