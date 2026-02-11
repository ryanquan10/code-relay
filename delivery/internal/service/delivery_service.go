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
	Token       string  `json:"token"`
	ProductCode string  `json:"product_code"`
	SKU         *string `json:"sku,omitempty"`
	Platform    string  `json:"platform"`
}

// DeliverProductRequest 按平台+内部产品码发货
type DeliverProductRequest struct {
	Token            string  `json:"token"`
	ProductInnerCode string  `json:"product_inner_code"`
	Platform         string  `json:"platform"`
	Group            *string `json:"group,omitempty"`
	ProductID        *int64  `json:"product_id,omitempty"`
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

	// 1. 根据 Platform、ProductCode 和可选的 SKU 查找 Product
	var product *entity.Product
	var err error
	if req.SKU != nil && *req.SKU != "" {
		product, err = s.repo.FindProductByPlatform(req.Platform, req.ProductCode, req.SKU)
	} else {
		products, err2 := s.repo.ListProductsByPlatform(req.Platform, req.ProductCode)
		if err2 != nil {
			return nil, fmt.Errorf("failed to list products: %w", err2)
		}
		if len(products) == 0 {
			return nil, fmt.Errorf("product not found for platform %s and product_code %s", req.Platform, req.ProductCode)
		}
		if len(products) == 1 {
			product = &products[0]
		} else {
			// 多个匹配时，要求 validity_days == 1
			var picked *entity.Product
			for i := range products {
				if products[i].ValidityDays == 1 {
					picked = &products[i]
					break
				}
			}
			if picked == nil {
				return nil, fmt.Errorf("multiple products matched for platform %s and product_code %s, but none has validity_days=1", req.Platform, req.ProductCode)
			}
			product = picked
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find product: %w", err)
	}
	if product == nil {
		if req.SKU != nil && *req.SKU != "" {
			return nil, fmt.Errorf("product not found for platform %s, product_code %s and sku %s", req.Platform, req.ProductCode, *req.SKU)
		}
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

// DeliverProduct 根据 token, platform, product_inner_code 发货，返回拼装文本
func (s *DeliveryService) DeliverProduct(req DeliverProductRequest) (string, error) {
	// 验证 token
	const validToken = "secret_token11234567"
	if req.Token != validToken {
		return "", fmt.Errorf("invalid token")
	}

	var product *entity.Product
	var err error

	// 如果有 group 参数，使用产品组逻辑
	if req.Group != nil && strings.TrimSpace(*req.Group) != "" {
		// 如果有 product_id，直接使用该产品
		if req.ProductID != nil && *req.ProductID > 0 {
			product, err = s.repo.FindProductByID(*req.ProductID)
			if err != nil {
				return "", fmt.Errorf("failed to find product by id: %w", err)
			}
			if product == nil {
				return "", fmt.Errorf("product not found for product_id %d", *req.ProductID)
			}
		} else {
			// 没有 product_id，按照产品组最小的来发货
			products, err := s.repo.FindProductsByGroup(*req.Group)
			if err != nil {
				return "", fmt.Errorf("failed to find products by group: %w", err)
			}
			if len(products) == 0 {
				return "", fmt.Errorf("no products found for group %s", *req.Group)
			}
			// 选择 ID 最小的产品
			product = &products[0]
			for i := range products {
				if products[i].ID < product.ID {
					product = &products[i]
				}
			}
		}
	} else {
		// 原来的逻辑：查商品（platform + product_code）
		product, err = s.repo.FindProductByInnerProductCode(req.ProductInnerCode)
		if err != nil {
			return "", fmt.Errorf("failed to find product: %w", err)
		}
		if product == nil {
			return "", fmt.Errorf("product not found for platform %s and product_code %s", req.Platform, req.ProductInnerCode)
		}
	}

	// 查可用账号（use_status=0）
	account, err := s.repo.FindAvailableAccount(product.ID)
	if err != nil {
		return "", fmt.Errorf("failed to find available account: %w", err)
	}
	if account == nil {
		return "", fmt.Errorf("no available account for product_id %d", product.ID)
	}

	// 标记为已使用，并设置 start_time
	if err := s.repo.UpdateAccountUseStatus(account.ID, 1); err != nil {
		return "", fmt.Errorf("failed to update account use_status: %w", err)
	}

	// 填充说明（支持 {TOKEN}/{token}、{account_email}/{account_password} 等）
	desc := fillDescription(product.Description, account.AccountEmail, account.AccountPassword, account.Token)

	var api string
	if product.DownStreamURL != nil {
		api = *product.DownStreamURL
	}

	var sb strings.Builder
	sb.WriteString("详细教程:")
	if desc != nil {
		sb.WriteString(*desc)
	}
	sb.WriteString("  api:")
	sb.WriteString(api)

	// 若 token 存在且未在描述中出现，则追加 token 字段
	if account.Token != nil && strings.TrimSpace(*account.Token) != "" {
		includeToken := true
		if desc != nil && strings.Contains(*desc, *account.Token) {
			includeToken = false
		}
		if includeToken {
			sb.WriteString("  ,token:")
			sb.WriteString(*account.Token)
		}
	}

	return sb.String(), nil
}

// fillDescription 用账号信息填充描述模板；若模板无占位符则在末尾追加“邮箱/密码”  token
func fillDescription(tpl *string, email string, password *string, token *string) *string {
	var base string
	if tpl != nil {
		base = *tpl
	}

	replaced := replacePlaceholders(base, email, password, token)
	if replaced == base {
		// 模板无可识别占位符：按实际存在的字段拼接，避免输出空的“邮箱/密码”标签
		trimmed := strings.TrimSpace(base)
		parts := make([]string, 0, 3)
		if strings.TrimSpace(email) != "" {
			parts = append(parts, fmt.Sprintf("邮箱: %s", email))
		}
		if password != nil && strings.TrimSpace(*password) != "" {
			parts = append(parts, fmt.Sprintf("密码: %s", *password))
		}
		if token != nil && strings.TrimSpace(*token) != "" {
			parts = append(parts, fmt.Sprintf("token: %s", *token))
		}

		// 如果既没有模板文本也没有可展示字段，则返回空描述（nil）
		if trimmed == "" && len(parts) == 0 {
			return nil
		}

		var sb strings.Builder
		if trimmed != "" {
			sb.WriteString(trimmed)
			if !strings.HasSuffix(trimmed, "\n") {
				sb.WriteString("\n")
			}
		}
		if len(parts) > 0 {
			sb.WriteString(strings.Join(parts, "   "))
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
		case "accountemail", "email", "accountemai": // 兼容 {account_emai}
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
