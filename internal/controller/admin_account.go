package controller

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"codex-relay/internal/repository"
	"crypto/rand"
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

	var accounts []entity.Account
	if err := db.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	account := &entity.Account{
		AccountEmail:    strings.TrimSpace(req.AccountEmail),
		AccountPassword: normalizeOptionalString(req.AccountPassword),
		Token:           token,
		Balance:         balance,
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
	productCache := make(map[int64]*entity.Product)
	sourceCache := make(map[int64]int64)

	for _, a := range req.Accounts {
		if strings.TrimSpace(a.Status) == "" {
			failed++
			continue
		}

		product, ok := productCache[a.ProductID]
		if !ok {
			fetched, err := getProductByID(db, a.ProductID)
			if err != nil {
				failed++
				continue
			}
			product = fetched
			productCache[a.ProductID] = product
		}

		sourceID, ok := sourceCache[a.ProductID]
		if !ok {
			fetchedSourceID, err := getDefaultSourceID(db, a.ProductID)
			if err != nil {
				failed++
				continue
			}
			sourceID = fetchedSourceID
			sourceCache[a.ProductID] = sourceID
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

		account := &entity.Account{
			AccountEmail:    strings.TrimSpace(a.AccountEmail),
			AccountPassword: normalizeOptionalString(a.AccountPassword),
			Token:           token,
			Balance:         balance,
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
		Balance float64 `json:"balance" binding:"required"`
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

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
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

type accountResponse struct {
	ID               uint64     `json:"id"`
	AccountEmail     string     `json:"account_email"`
	Token            *string    `json:"token"`
	ProductID        int64      `json:"product_id"`
	SourceID         int64      `json:"source_id"`
	UserID           *uint64    `json:"user_id"`
	Status           string     `json:"status"`
	ExpireDate       *time.Time `json:"expire_date"`
	Balance          float64    `json:"balance"`
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
		ExpireDate:       account.ExpireDate,
		Balance:          account.Balance,
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
