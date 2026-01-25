package controller

import (
	"codex-relay/pkg/utils"
	"codex-relay/internal/mysql"
	"codex-relay/internal/repository"
	"codex-relay/pkg/entity"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListAccounts 获取账号列表
func ListAccounts(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	productIDStr := strings.TrimSpace(c.Query("product_id"))
	var accounts []entity.Account
	if productIDStr != "" {
		pid, err := strconv.ParseInt(productIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
			return
		}
		list, err := repository.NewAccountRepository().GetByProductId(pid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		accounts = list
	} else {
		if err := db.Find(&accounts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	response := make([]accountResponse, 0, len(accounts))
	for i := range accounts {
		response = append(response, newAccountResponse(&accounts[i]))
	}
	c.JSON(http.StatusOK, response)
}

// GetAccountByToken 根据 Token 查询账号
func GetAccountByToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	repo := repository.NewAccountRepository()
	account, err := repo.GetByToken(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if account == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	c.JSON(http.StatusOK, newAccountResponse(account))
}

// CreateAccount 创建账号
func CreateAccount(c *gin.Context) {
	var req struct {
		AccountEmail    string     `json:"account_email"`
		AccountPassword *string    `json:"account_password"`
		Token           *string    `json:"token"`
		Balance         *float64   `json:"balance"`
		UsedBalance     *float64   `json:"used_balance"`
		UseStatus       *int       `json:"use_status"`
		Status          string     `json:"status" binding:"required"`
		ProductID       int64      `json:"product_id" binding:"required"`
		UserID          *uint64    `json:"user_id"`
		ExpireDate      *time.Time `json:"expire_date"`
		Remark          *string    `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.Status) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status cannot be empty"})
		return
	}

	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	product, err := getProductByID(db, req.ProductID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	sourceID, err := getDefaultSourceID(db, product.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product has no account sources"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	token := normalizeOptionalString(req.Token)
	if token == nil {
		generatedToken, err := buildDefaultToken(product.ProductCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		token = &generatedToken
	}

	balance := product.DefaultBalance
	if req.Balance != nil {
		balance = *req.Balance
	}

	usedBalance := 0.0
	if req.UsedBalance != nil {
		usedBalance = *req.UsedBalance
	}

	useStatus := 0
	if req.UseStatus != nil {
		useStatus = *req.UseStatus
	}

	account := &entity.Account{
		AccountEmail:    strings.TrimSpace(req.AccountEmail),
		AccountPassword: normalizeOptionalString(req.AccountPassword),
		Token:           token,
		Balance:         balance,
		UsedBalance:     usedBalance,
		UseStatus:       useStatus,
		Status:          strings.TrimSpace(req.Status),
		ProductID:       req.ProductID,
		SourceID:        sourceID,
		UserID:          req.UserID,
		ExpireDate:      req.ExpireDate,
		Remark:          normalizeOptionalString(req.Remark),
	}

	repo := repository.NewAccountRepository()
	if err := repo.Create(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": account.ID})
}

// BatchCreateAccounts 批量创建账号
func BatchCreateAccounts(c *gin.Context) {
    // Support two modes: structured accounts[] or raw text + field mapping
    var req struct {
        Text         string   `json:"text"`
        FieldKeys    []string `json:"field_keys"`
        FieldKeysStr string   `json:"field_keys_str"`
        // Defaults for text mode
        ProductID    int64    `json:"product_id"`
        Status       string   `json:"status"`
        Accounts     []struct {
            AccountEmail    string     `json:"account_email"`
            AccountPassword *string    `json:"account_password"`
            Token           *string    `json:"token"`
            Balance         *float64   `json:"balance"`
            UsedBalance     *float64   `json:"used_balance"`
            UseStatus       *int       `json:"use_status"`
            Status          string     `json:"status" binding:"required"`
            ProductID       int64      `json:"product_id" binding:"required"`
            UserID          *uint64    `json:"user_id"`
            ExpireDate      *time.Time `json:"expire_date"`
            Remark          *string    `json:"remark"`
        } `json:"accounts"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    db := mysql.DB()
    if db == nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
        return
    }

    repo := repository.NewAccountRepository()
    success := 0
    failed := 0

    // Helper: normalize key to compare across camel/snake
    normalizeKey := func(s string) string {
        s = strings.ToLower(strings.TrimSpace(s))
        s = strings.ReplaceAll(s, "_", "")
        return s
    }
    getVal := func(m map[string]string, want string) (string, bool) {
        wantN := normalizeKey(want)
        for k, v := range m {
            if normalizeKey(k) == wantN {
                return strings.TrimSpace(v), true
            }
        }
        return "", false
    }

    existsEmail := func(email string) bool {
        if strings.TrimSpace(email) == "" { return false }
        a, err := repo.GetByEmail(email)
        return err == nil && a != nil
    }
    existsToken := func(token string) bool {
        if strings.TrimSpace(token) == "" { return false }
        a, err := repo.GetByToken(token)
        return err == nil && a != nil
    }

    seenEmails := make(map[string]struct{})
    seenTokens := make(map[string]struct{})

    // If text mode is provided and accounts are empty, parse via utils
    if len(req.Accounts) == 0 && strings.TrimSpace(req.Text) != "" {
        fieldKeys := req.FieldKeys
        if len(fieldKeys) == 0 {
            if strings.TrimSpace(req.FieldKeysStr) != "" {
                for _, p := range strings.Split(req.FieldKeysStr, ",") {
                    p = strings.TrimSpace(p)
                    if p != "" { fieldKeys = append(fieldKeys, p) }
                }
            }
        }
        if len(fieldKeys) == 0 {
            // default order: first email, second password, then token and balance if present
            fieldKeys = []string{"accountEmail", "accountPassword", "token", "balance"}
        }
        // Require defaults for product and status in text mode
        if req.ProductID == 0 {
            c.JSON(http.StatusBadRequest, gin.H{"error": "product_id required for text import"})
            return
        }
        if strings.TrimSpace(req.Status) == "" {
            req.Status = "active"
        }

        extracted := utils.ExtractAccounts(req.Text, fieldKeys)
        for _, m := range extracted {
            email, _ := getVal(m, "accountEmail")
            if email == "" { email, _ = getVal(m, "account_email") }
            password, _ := getVal(m, "accountPassword")
            if password == "" { password, _ = getVal(m, "account_password") }
            tokenStr, _ := getVal(m, "token")
            balanceStr, _ := getVal(m, "balance")
            usedBalStr, _ := getVal(m, "used_balance")
            if usedBalStr == "" { usedBalStr, _ = getVal(m, "usedBalance") }
            useStatusStr, _ := getVal(m, "use_status")
            if useStatusStr == "" { useStatusStr, _ = getVal(m, "useStatus") }
            prodStr, _ := getVal(m, "product_id")
            if prodStr == "" { prodStr, _ = getVal(m, "productId") }
            statusStr, _ := getVal(m, "status")

            // determine product
            productID := req.ProductID
            if p, err := strconv.ParseInt(prodStr, 10, 64); err == nil && p > 0 {
                productID = p
            }

            product, err := getProductByID(db, productID)
            if err != nil {
                failed++
                continue
            }
            sourceID, err := getDefaultSourceID(db, product.ID)
            if err != nil {
                failed++
                continue
            }

            // normalize values
            var token *string
            t := strings.TrimSpace(tokenStr)
            if t == "" {
                gen, err := buildDefaultToken(product.ProductCode)
                if err == nil { token = &gen }
            } else {
                token = &t
            }

            var pass *string
            if strings.TrimSpace(password) != "" {
                p := strings.TrimSpace(password)
                pass = &p
            }

            balance := product.DefaultBalance
            if b, err := strconv.ParseFloat(strings.TrimSpace(balanceStr), 64); err == nil {
                balance = b
            }
            usedBalance := 0.0
            if ub, err := strconv.ParseFloat(strings.TrimSpace(usedBalStr), 64); err == nil {
                usedBalance = ub
            }
            useStatus := 0
            if us, err := strconv.Atoi(strings.TrimSpace(useStatusStr)); err == nil {
                useStatus = us
            }
            status := strings.TrimSpace(statusStr)
            if status == "" { status = strings.TrimSpace(req.Status) }
            if status == "" { status = "active" }

            email = strings.TrimSpace(email)
            if email != "" {
                if _, ok := seenEmails[email]; ok || existsEmail(email) {
                    failed++
                    continue
                }
            }
            if token != nil && strings.TrimSpace(*token) != "" {
                tok := strings.TrimSpace(*token)
                if _, ok := seenTokens[tok]; ok || existsToken(tok) {
                    failed++
                    continue
                }
            }

            account := &entity.Account{
                AccountEmail:    email,
                AccountPassword: pass,
                Token:           token,
                Balance:         balance,
                UsedBalance:     usedBalance,
                UseStatus:       useStatus,
                Status:          status,
                ProductID:       productID,
                SourceID:        sourceID,
            }

            if err := repo.Create(account); err != nil {
                failed++
            } else {
                success++
                if email != "" { seenEmails[email] = struct{}{} }
                if token != nil && strings.TrimSpace(*token) != "" { seenTokens[strings.TrimSpace(*token)] = struct{}{} }
            }
        }

        c.JSON(http.StatusOK, gin.H{"success": success, "failed": failed})
        return
    }

    // Structured accounts path (original)
    for _, a := range req.Accounts {
        if strings.TrimSpace(a.Status) == "" {
            failed++
            continue
        }
        product, err := getProductByID(db, a.ProductID)
        if err != nil {
            failed++
            continue
        }
        sourceID, err := getDefaultSourceID(db, product.ID)
        if err != nil {
            failed++
            continue
        }
        token := normalizeOptionalString(a.Token)
        if token == nil {
            generatedToken, err := buildDefaultToken(product.ProductCode)
            if err != nil {
                failed++
                continue
            }
            token = &generatedToken
        }
        email := strings.TrimSpace(a.AccountEmail)
        if email != "" {
            if _, ok := seenEmails[email]; ok || existsEmail(email) {
                failed++
                continue
            }
        }
        if token != nil && strings.TrimSpace(*token) != "" {
            tok := strings.TrimSpace(*token)
            if _, ok := seenTokens[tok]; ok || existsToken(tok) {
                failed++
                continue
            }
        }
        balance := product.DefaultBalance
        if a.Balance != nil { balance = *a.Balance }
        usedBalance := 0.0
        if a.UsedBalance != nil { usedBalance = *a.UsedBalance }
        useStatus := 0
        if a.UseStatus != nil { useStatus = *a.UseStatus }

        account := &entity.Account{
            AccountEmail:    email,
            AccountPassword: normalizeOptionalString(a.AccountPassword),
            Token:           token,
            Balance:         balance,
            UsedBalance:     usedBalance,
            UseStatus:       useStatus,
            Status:          strings.TrimSpace(a.Status),
            ProductID:       a.ProductID,
            SourceID:        sourceID,
            UserID:          a.UserID,
            ExpireDate:      a.ExpireDate,
            Remark:          normalizeOptionalString(a.Remark),
        }
        if err := repo.Create(account); err != nil {
            failed++
        } else {
            success++
            if email != "" { seenEmails[email] = struct{}{} }
            if token != nil && strings.TrimSpace(*token) != "" { seenTokens[strings.TrimSpace(*token)] = struct{}{} }
        }
    }

    c.JSON(http.StatusOK, gin.H{"success": success, "failed": failed})
}

// UpdateAccountBalance 更新账户余额
func UpdateAccountBalance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		Balance     float64  `json:"balance" binding:"required"`
		UsedBalance *float64 `json:"used_balance"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := repository.NewAccountRepository()
	if err := repo.UpdateBalance(id, req.Balance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.UsedBalance != nil {
		if err := repo.UpdateUsedBalance(id, *req.UsedBalance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// UpdateAccount 通用更新（含 use_status）
func UpdateAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req struct {
		AccountEmail    *string    `json:"account_email"`
		ProductID       *int64     `json:"product_id"`
		AccountPassword *string    `json:"account_password"`
		Token           *string    `json:"token"`
		Balance         *float64   `json:"balance"`
		UsedBalance     *float64   `json:"used_balance"`
		Status          *string    `json:"status"`
		ExpireDate      *time.Time `json:"expire_date"`
		Remark          *string    `json:"remark"`
		UseStatus       *int       `json:"use_status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	repo := repository.NewAccountRepository()
	account, err := repo.GetById(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if account == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if req.AccountEmail != nil {
		account.AccountEmail = strings.TrimSpace(*req.AccountEmail)
	}
	if req.AccountPassword != nil {
		account.AccountPassword = normalizeOptionalString(req.AccountPassword)
	}
	if req.Token != nil {
		account.Token = normalizeOptionalString(req.Token)
	}
	if req.Balance != nil {
		account.Balance = *req.Balance
	}
	if req.UsedBalance != nil {
		account.UsedBalance = *req.UsedBalance
	}
	if req.Status != nil {
		account.Status = strings.TrimSpace(*req.Status)
	}
	if req.ExpireDate != nil {
		account.ExpireDate = req.ExpireDate
	}
	if req.Remark != nil {
		account.Remark = normalizeOptionalString(req.Remark)
	}
	if req.UseStatus != nil {
		account.UseStatus = *req.UseStatus
	}

	if req.ProductID != nil && account.ProductID != *req.ProductID {
		product, err := getProductByID(db, *req.ProductID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "product not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sourceID, err := getDefaultSourceID(db, product.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "product has no account sources"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		account.ProductID = *req.ProductID
		account.SourceID = sourceID
	}
	if err := repo.Update(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, newAccountResponse(account))
}

// DeleteAccount 删除账号
func DeleteAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	repo := repository.NewAccountRepository()
	if err := repo.DeleteById(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// BatchDeleteAccounts 批量删除账号
func BatchDeleteAccounts(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids cannot be empty"})
		return
	}

	repo := repository.NewAccountRepository()
	if err := repo.DeleteByIds(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "deleted",
		"count":   len(req.IDs),
	})
}

type accountResponse struct {
	ID               uint64     `json:"id"`
	AccountEmail     string     `json:"account_email"`
	AccountPassword  *string    `json:"account_password"`
	Token            *string    `json:"token"`
	ProductID        int64      `json:"product_id"`
	SourceID         int64      `json:"source_id"`
	UserID           *uint64    `json:"user_id"`
	Status           string     `json:"status"`
	UseStatus        int        `json:"use_status"`
	ExpireDate       *time.Time `json:"expire_date"`
	Balance          float64    `json:"balance"`
	UsedBalance      float64    `json:"used_balance"`
	LastRechargeTime *time.Time `json:"last_recharge_time"`
	Remark           *string    `json:"remark"`
	Version          int        `json:"version"`
	CreateTime       time.Time  `json:"create_time"`
	UpdateTime       time.Time  `json:"update_time"`
}

func newAccountResponse(account *entity.Account) accountResponse {
	return accountResponse{
		ID:               account.ID,
		AccountEmail:     account.AccountEmail,
		AccountPassword:  account.AccountPassword,
		Token:            account.Token,
		ProductID:        account.ProductID,
		SourceID:         account.SourceID,
		UserID:           account.UserID,
		Status:           account.Status,
		UseStatus:        account.UseStatus,
		ExpireDate:       account.ExpireDate,
		Balance:          account.Balance,
		UsedBalance:      account.UsedBalance,
		LastRechargeTime: account.LastRechargeTime,
		Remark:           account.Remark,
		Version:          account.Version,
		CreateTime:       account.CreateTime,
		UpdateTime:       account.UpdateTime,
	}
}

func buildDefaultToken(prefix string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(prefix), "-")
	if trimmed == "" {
		return "", errors.New("product_code is empty")
	}
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return trimmed + "-" + hex.EncodeToString(randomBytes), nil
}

func getProductByID(db *gorm.DB, productID int64) (*entity.Product, error) {
	var product entity.Product
	if err := db.First(&product, productID).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func getDefaultSourceID(db *gorm.DB, productID int64) (int64, error) {
	var mapping entity.AccountSourceProcut
	if err := db.Where("product_id = ?", productID).Order("weight desc, id asc").First(&mapping).Error; err != nil {
		return 0, err
	}
	return mapping.SourceID, nil
}

// ExportAccounts 批量导出账号为CSV
// 支持按 ids（逗号分隔）或按 product_id 筛选；都未提供时导出全部
func ExportAccounts(c *gin.Context) {
	db := mysql.DB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	idsParam := strings.TrimSpace(c.Query("ids"))
	productIDStr := strings.TrimSpace(c.Query("product_id"))

	var accounts []entity.Account

	if idsParam != "" {
		parts := strings.Split(idsParam, ",")
		ids := make([]uint64, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseUint(p, 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ids"})
				return
			}
			ids = append(ids, id)
		}
		if err := db.Where("id IN ?", ids).Find(&accounts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if productIDStr != "" {
		pid, err := strconv.ParseInt(productIDStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product_id"})
			return
		}
		if err := db.Where("product_id = ?", pid).Find(&accounts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := db.Find(&accounts).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	filename := "accounts_" + time.Now().Format("20060102150405") + ".csv"
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	_ = w.Write([]string{
		"id", "product_id", "account_email", "token", "balance", "used_balance", "status", "use_status", "expire_date", "create_time",
	})

	for _, a := range accounts {
		expire := ""
		if a.ExpireDate != nil {
			expire = a.ExpireDate.Format(time.RFC3339)
		}
		_ = w.Write([]string{
			strconv.FormatUint(a.ID, 10),
			strconv.FormatInt(a.ProductID, 10),
			a.AccountEmail,
			valueOrEmpty(a.Token),
			formatFloat(a.Balance),
			formatFloat(a.UsedBalance),
			a.Status,
			strconv.Itoa(a.UseStatus),
			expire,
			a.CreateTime.Format(time.RFC3339),
		})
	}
}

func valueOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
