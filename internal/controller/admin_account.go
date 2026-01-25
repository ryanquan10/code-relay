package controller

import (
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
	var req struct {
		Accounts []struct {
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
		} `json:"accounts" binding:"required"`
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

		balance := product.DefaultBalance
		if a.Balance != nil {
			balance = *a.Balance
		}
		usedBalance := 0.0
		if a.UsedBalance != nil {
			usedBalance = *a.UsedBalance
		}
		useStatus := 0
		if a.UseStatus != nil {
			useStatus = *a.UseStatus
		}

		account := &entity.Account{
			AccountEmail:    strings.TrimSpace(a.AccountEmail),
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
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"failed":  failed,
	})
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
