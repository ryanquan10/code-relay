package service

import (
	"fmt"
	"strings"

	"codex-relay/internal/mysql"
	"codex-relay/pkg/entity"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// 默认值来自原先写死计价（USD）：in = 1.5925 / 1M，out = 14 / 1M
	DefaultInTokenUnitPrice  = 1.5925
	DefaultOutTokenUnitPrice = 14.0
	DefaultTokenUnit         = int64(1_000_000)
	DefaultPricingUnit       = "US"

	approxInputRatio  = 0.70
	approxOutputRatio = 0.30
)

type PricingService struct{}

func NewPricingService() *PricingService {
	return &PricingService{}
}

func normalizePricingUnit(unit string) string {
	u := strings.ToUpper(strings.TrimSpace(unit))
	switch u {
	case "", "USD", "US":
		return "US"
	case "RMB", "CNY":
		return "RMB"
	default:
		return u
	}
}

func DefaultPricingForAccountType(accountType string) entity.Pricing {
	return entity.Pricing{
		AccountType:       strings.TrimSpace(accountType),
		InTokenUnitPrice:  DefaultInTokenUnitPrice,
		OutTokenUnitPrice: DefaultOutTokenUnitPrice,
		TokenUnit:         DefaultTokenUnit,
		Unit:              DefaultPricingUnit,
	}
}

func (s *PricingService) ensureDB() (*gorm.DB, error) {
	db := mysql.DB()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return db, nil
}

// EnsureDefaultsFromProducts 根据 product.account_type 自动补齐 pricing 默认数据
func (s *PricingService) EnsureDefaultsFromProducts() error {
	db, err := s.ensureDB()
	if err != nil {
		return err
	}

	var accountTypes []string
	if err := db.Model(&entity.Product{}).
		Where("TRIM(account_type) <> ''").
		Group("TRIM(account_type)").
		Order("TRIM(account_type) ASC").
		Pluck("TRIM(account_type)", &accountTypes).Error; err != nil {
		return fmt.Errorf("failed to query distinct account_type from product: %w", err)
	}

	for _, accountType := range accountTypes {
		if err := s.ensurePricingForAccountType(db, accountType); err != nil {
			return err
		}
	}
	return nil
}

func (s *PricingService) ensurePricingForAccountType(db *gorm.DB, accountType string) error {
	accountType = strings.TrimSpace(accountType)
	if accountType == "" {
		return nil
	}

	defaultPricing := DefaultPricingForAccountType(accountType)
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account_type"}},
		DoNothing: true,
	}).Create(&defaultPricing).Error; err != nil {
		return fmt.Errorf("failed to ensure pricing for account_type %s: %w", accountType, err)
	}
	return nil
}

func (s *PricingService) ListByAccountType() ([]entity.Pricing, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	if err := s.EnsureDefaultsFromProducts(); err != nil {
		return nil, err
	}

	var items []entity.Pricing
	if err := db.Table("pricing AS pr").
		Select("pr.*").
		Joins(`JOIN (
			SELECT TRIM(account_type) AS account_type
			FROM product
			WHERE TRIM(account_type) <> ''
			GROUP BY TRIM(account_type)
		) pa ON pa.account_type = pr.account_type`).
		Order("pr.account_type ASC").
		Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to query pricing list by product account_type: %w", err)
	}
	for i := range items {
		items[i].Unit = normalizePricingUnit(items[i].Unit)
	}
	return items, nil
}

func (s *PricingService) GetOrInitByAccountType(accountType string) (*entity.Pricing, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}

	accountType = strings.TrimSpace(accountType)
	if accountType == "" {
		p := DefaultPricingForAccountType("")
		return &p, nil
	}

	var pricing entity.Pricing
	if err := db.Where("account_type = ?", accountType).First(&pricing).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to query pricing by account_type: %w", err)
		}
		if err := s.ensurePricingForAccountType(db, accountType); err != nil {
			return nil, err
		}
		if err := db.Where("account_type = ?", accountType).First(&pricing).Error; err != nil {
			return nil, fmt.Errorf("failed to query pricing after init: %w", err)
		}
	}
	pricing.Unit = normalizePricingUnit(pricing.Unit)
	return &pricing, nil
}

// ResolveAccountTypeByCustomerToken 根据 account.token -> product.account_type 解析计价维度
func (s *PricingService) ResolveAccountTypeByCustomerToken(customerToken string) (string, error) {
	db, err := s.ensureDB()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(customerToken)
	if token == "" {
		return "", nil
	}

	var row struct {
		AccountType string `gorm:"column:account_type"`
	}
	tx := db.Table("account AS a").
		Select("p.account_type").
		Joins("LEFT JOIN product p ON p.id = a.product_id").
		Where("a.token = ?", token).
		Limit(1).
		Scan(&row)
	if tx.Error != nil {
		return "", fmt.Errorf("failed to resolve account_type by token: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return "", nil
	}
	return strings.TrimSpace(row.AccountType), nil
}

func (s *PricingService) CalculateConsumeByCustomerToken(customerToken string, tokens uint64, inTokens uint64, outTokens uint64) (float64, *entity.Pricing, string, error) {
	accountType, err := s.ResolveAccountTypeByCustomerToken(customerToken)
	if err != nil {
		return 0, nil, "", err
	}

	pricing, err := s.GetOrInitByAccountType(accountType)
	if err != nil {
		return 0, nil, accountType, err
	}

	consume := CalculateConsumeByPricing(tokens, inTokens, outTokens, *pricing)
	return consume, pricing, accountType, nil
}

func CalculateConsumeByPricing(tokens uint64, inTokens uint64, outTokens uint64, pricing entity.Pricing) float64 {
	tokenUnit := pricing.TokenUnit
	if tokenUnit <= 0 {
		tokenUnit = DefaultTokenUnit
	}

	inPrice := pricing.InTokenUnitPrice
	outPrice := pricing.OutTokenUnitPrice

	var useInTokens float64
	var useOutTokens float64

	if inTokens > 0 || outTokens > 0 {
		useInTokens = float64(inTokens)
		useOutTokens = float64(outTokens)
	} else {
		useInTokens = float64(tokens) * approxInputRatio
		useOutTokens = float64(tokens) * approxOutputRatio
	}

	return (useInTokens*inPrice + useOutTokens*outPrice) / float64(tokenUnit)
}
