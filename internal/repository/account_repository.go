package repository

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AccountRepository 处理 Account 数据的存储和读取
type AccountRepository struct{}

// NewAccountRepository 创建新的 AccountRepository 实例
func NewAccountRepository() *AccountRepository {
	return &AccountRepository{}
}

// GetByToken 根据 Token 查询 Account
func (r *AccountRepository) GetByToken(token string) (*entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var account entity.Account
	err := db.Where("token = ?", token).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account by token: %w", err)
	}

	return &account, nil
}

// GetById 根据 ID 获取 Account
func (r *AccountRepository) GetById(id uint64) (*entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var account entity.Account
	err := db.First(&account, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account by id %d: %w", id, err)
	}

	return &account, nil
}

// GetByEmail 根据 Email 查询 Account
func (r *AccountRepository) GetByEmail(email string) (*entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var account entity.Account
	err := db.Where("account_email = ?", email).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get account by email: %w", err)
	}

	return &account, nil
}

// UpdateBalance 更新账户余额
func (r *AccountRepository) UpdateBalance(id uint64, balance float64) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	err := db.Model(&entity.Account{}).
		Where("id = ?", id).
		Update("balance", balance).Error

	if err != nil {
		return fmt.Errorf("failed to update balance for account %d: %w", id, err)
	}

	return nil
}

// DeductBalance 扣减账户余额（事务安全）
func (r *AccountRepository) DeductBalance(id uint64, amount float64) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	// 使用事务和行锁确保扣减安全
	err := db.Transaction(func(tx *gorm.DB) error {
		var account entity.Account
		// 加行锁 (FOR UPDATE)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&account, id).Error; err != nil {
			return fmt.Errorf("failed to lock account: %w", err)
		}

		// 检查余额是否足够
		if account.Balance < amount {
			return fmt.Errorf("insufficient balance: has %.2f, need %.2f", account.Balance, amount)
		}

		// 扣减余额
		newBalance := account.Balance - amount
		if err := tx.Model(&entity.Account{}).
			Where("id = ?", id).
			Update("balance", newBalance).Error; err != nil {
			return fmt.Errorf("failed to deduct balance: %w", err)
		}

		return nil
	})

	return err
}

// IncrementUsedBalance 增加已使用余额（不减少总余额）
// 使用场景：记录用户消费，累加到 used_balance，但不改变 balance
// 余额检查规则：当 used_balance > balance 时，拒绝请求
func (r *AccountRepository) IncrementUsedBalance(id uint64, amount float64) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	// 使用事务和行锁确保更新安全
	err := db.Transaction(func(tx *gorm.DB) error {
		var account entity.Account
		// 加行锁 (FOR UPDATE)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&account, id).Error; err != nil {
			return fmt.Errorf("failed to lock account: %w", err)
		}

		// 增加 used_balance
		newUsedBalance := account.UsedBalance + amount
		if err := tx.Model(&entity.Account{}).
			Where("id = ?", id).
			Update("used_balance", newUsedBalance).Error; err != nil {
			return fmt.Errorf("failed to increment used_balance: %w", err)
		}

		return nil
	})

	return err
}

// UpdateUsedBalance 直接设置已使用余额（用于初始化或修正数据）
// 注意：正常情况下应该使用 IncrementUsedBalance 来累加，此方法仅用于管理员手动修正
func (r *AccountRepository) UpdateUsedBalance(id uint64, usedBalance float64) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	// 使用事务和行锁确保更新安全
	err := db.Transaction(func(tx *gorm.DB) error {
		var account entity.Account
		// 加行锁 (FOR UPDATE)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&account, id).Error; err != nil {
			return fmt.Errorf("failed to lock account: %w", err)
		}

		// 直接设置 used_balance
		if err := tx.Model(&entity.Account{}).
			Where("id = ?", id).
			Update("used_balance", usedBalance).Error; err != nil {
			return fmt.Errorf("failed to update used_balance: %w", err)
		}

		return nil
	})

	return err
}

// Create 创建新的 Account
func (r *AccountRepository) Create(account *entity.Account) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.Create(account).Error; err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

// Update 更新 Account
func (r *AccountRepository) Update(account *entity.Account) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.Save(account).Error; err != nil {
		return fmt.Errorf("failed to update account: %w", err)
	}

	return nil
}

// DeleteById 删除指定 ID 的 Account
func (r *AccountRepository) DeleteById(id uint64) error {
	db := mysql.DB()
	if db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	err := db.Delete(&entity.Account{}, id).Error
	if err != nil {
		return fmt.Errorf("failed to delete account %d: %w", id, err)
	}

	return nil
}

// GetByUserId 根据 UserID 查询该用户的所有账户
func (r *AccountRepository) GetByUserId(userId uint64) ([]entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var accounts []entity.Account
	err := db.Where("user_id = ?", userId).Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by user_id %d: %w", userId, err)
	}

	return accounts, nil
}

// GetByStatus 根据状态查询账户列表
func (r *AccountRepository) GetByStatus(status string) ([]entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var accounts []entity.Account
	err := db.Where("status = ?", status).Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by status %s: %w", status, err)
	}

	return accounts, nil
}

// GetByProductId 根据产品ID查询账户列表
func (r *AccountRepository) GetByProductId(productId int64) ([]entity.Account, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database connection is not initialized")
	}

	var accounts []entity.Account
	err := db.Where("product_id = ?", productId).Find(&accounts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts by product_id %d: %w", productId, err)
	}

	return accounts, nil
}
