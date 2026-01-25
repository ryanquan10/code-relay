package service

import (
	"codex-relay/pkg/entity"
	"delivery/internal/repository"
	"fmt"
	"regexp"
	"strings"
)

// DeliveryService 处理账号交付的业务逻辑
type DeliveryService struct {
	repo *repository.DeliveryRepository
}

// NewDeliveryService 创建新的 DeliveryService 实例
func NewDeliveryService(repo *repository.DeliveryRepository) *DeliveryService {
	return &DeliveryService{repo: repo}
}

// GetAccountRequest 获取账号的请求参数
type GetAccountRequest struct {
	Token       string `json:"token"`
	ProductCode string `json:"product_code"`
	Platform    string `json:"platform"`
}

// AccountData 账号数据响应
type AccountData struct {
	AccountEmail    string  `json:"account_email"`
	AccountPassword *string `json:"account_password"`
	Token           *string `json:"token"`
	Description     *string `json:"description"`
	Balance         float64 `json:"balance"`
}

// GetAccountResponse 获取账号的响应
type GetAccountResponse struct {
	Data *AccountData `json:"data"`
}

// GetAccount 根据 token, ProductCode, Platform 获取可用账号
func (s *DeliveryService) GetAccount(req GetAccountRequest) (*AccountData, error) {
	// 验证 token
	const validToken = "secret_token11234567"
	if req.Token != validToken {
		return nil, fmt.Errorf("invalid token")
	}

	// 1. 根据 Platform 和 ProductCode 查找 Product
	var product *entity.Product
	product, err := s.repo.FindProductByPlatform(req.Platform, req.ProductCode)
	if err != nil {
		return nil, fmt.Errorf("failed to find product: %w", err)
	}
	if product == nil {
		return nil, fmt.Errorf("product not found for platform %s and product_code %s", req.Platform, req.ProductCode)
	}

	// 2. 根据 Product.ID 查找可用的 Account
	var account *entity.Account
	account, err = s.repo.FindAvailableAccount(product.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find available account: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("no available account for product_id %d", product.ID)
	}

	// 3. 更新 Account 的 UseStatus 为 1
	err = s.repo.UpdateAccountUseStatus(account.ID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to update account use_status: %w", err)
	}

	// 4. 生成描述信息（支持 {account_email}/{accountEmail}、{account_password}/{accountPassword} 等大小写/驼峰占位符）
	desc := fillDescription(product.Description, account.AccountEmail, account.AccountPassword, account.Token)

	// 返回指定字段
	return &AccountData{
		AccountEmail:    account.AccountEmail,
		AccountPassword: account.AccountPassword,
		Token:           account.Token,
		Description:     desc,
		Balance:         account.Balance,
	}, nil
}

// fillDescription 用账号信息填充描述模板；若模板无占位符则在末尾追加“邮箱/密码”  token
func fillDescription(tpl *string, email string, password *string, token *string) *string {
	var base string
	if tpl != nil {
		base = *tpl
	}

	replaced := replacePlaceholders(base, email, password, token)
	if replaced == base {
		// 模板无可识别占位符，则在末尾拼接账号信息
		pwd := ""
		if password != nil {
			pwd = *password
		}
		tk := ""
		if token != nil {
			tk = *token
		}

		var sb strings.Builder
		trimmed := strings.TrimSpace(base)
		if trimmed != "" {
			sb.WriteString(trimmed)
			if !strings.HasSuffix(trimmed, "\n") {
				sb.WriteString("\n")
			}
		}

		sb.WriteString(fmt.Sprintf("邮箱: %s   密码: %s", email, pwd))
		if tk != "" {
			sb.WriteString(fmt.Sprintf("   token: %s", tk))
		}

		s := sb.String()
		return &s
	}
	return &replaced
}

// replacePlaceholders 将 { ... } 中的键做归一化后匹配替换（支持 token）
func replacePlaceholders(tpl string, email string, password *string, token *string) string {
	re := regexp.MustCompile(`\{\s*([^{}]+?)\s*\}`)
	return re.ReplaceAllStringFunc(tpl, func(s string) string {
		sub := re.FindStringSubmatch(s)
		if len(sub) < 2 {
			return s
		}
		key := sub[1]

		// 归一化：转小写、去除下划线、连字符、空格
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, "_", "")
		key = strings.ReplaceAll(key, "-", "")
		key = strings.ReplaceAll(key, " ", "")

		switch key {
		case "accountemail", "email":
			return email
		case "accountpassword", "password":
			if password != nil {
				return *password
			}
			return ""
		case "accounttoken", "token":
			if token != nil {
				return *token
			}
			return ""
		default:
			return s
		}
	})
}
