package controller

import (
	"codex-relay/internal/entity"
	"codex-relay/internal/mysql"
	"codex-relay/internal/repository"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
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

	// 转换为前端需要的格式
	type AccountResponse struct {
		ID           uint64  `json:"id"`
		UserID       uint64  `json:"user_id"`
		ProductID    int64   `json:"product_id"`
		AccountEmail string  `json:"account_email"`
		Token        string  `json:"token"`
		Balance      float64 `json:"balance"`
		Status       string  `json:"status"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
	}

	var response []AccountResponse
	for _, a := range accounts {
		token := ""
		if a.Token != nil {
			token = *a.Token
		}
		userID := uint64(0)
		if a.UserID != nil {
			userID = *a.UserID
		}
		response = append(response, AccountResponse{
			ID:           a.ID,
			UserID:       userID,
			ProductID:    a.ProductID,
			AccountEmail: a.AccountEmail,
			Token:        token,
			Balance:      a.Balance,
			Status:       a.Status,
			CreatedAt:    a.CreateTime.Format("2006-01-02 15:04:05"),
			UpdatedAt:    a.UpdateTime.Format("2006-01-02 15:04:05"),
		})
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

	tokenStr := ""
	if account.Token != nil {
		tokenStr = *account.Token
	}
	userID := uint64(0)
	if account.UserID != nil {
		userID = *account.UserID
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            account.ID,
		"user_id":       userID,
		"product_id":    account.ProductID,
		"account_email": account.AccountEmail,
		"token":         tokenStr,
		"balance":       account.Balance,
		"status":        account.Status,
		"created_at":    account.CreateTime.Format("2006-01-02 15:04:05"),
		"updated_at":    account.UpdateTime.Format("2006-01-02 15:04:05"),
	})
}

// CreateAccount 创建账号
func CreateAccount(c *gin.Context) {
	var req struct {
		AccountEmail string  `json:"account_email" binding:"required"`
		Token        string  `json:"token" binding:"required"`
		Balance      float64 `json:"balance"`
		Status       string  `json:"status"`
		ProductID    int64   `json:"product_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := "active"
	if req.Status != "" {
		status = req.Status
	}

	productID := int64(1)
	if req.ProductID > 0 {
		productID = req.ProductID
	}

	account := &entity.Account{
		AccountEmail: req.AccountEmail,
		Token:        &req.Token,
		Balance:      req.Balance,
		Status:       status,
		ProductID:    productID,
		SourceID:     1,
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
			AccountEmail string  `json:"account_email" binding:"required"`
			Token        string  `json:"token" binding:"required"`
			Balance      float64 `json:"balance"`
			Status       string  `json:"status"`
			ProductID    int64   `json:"product_id"`
		} `json:"accounts" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := repository.NewAccountRepository()
	success := 0
	failed := 0

	for _, a := range req.Accounts {
		status := "active"
		if a.Status != "" {
			status = a.Status
		}

		productID := int64(1)
		if a.ProductID > 0 {
			productID = a.ProductID
		}

		account := &entity.Account{
			AccountEmail: a.AccountEmail,
			Token:        &a.Token,
			Balance:      a.Balance,
			Status:       status,
			ProductID:    productID,
			SourceID:     1,
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
