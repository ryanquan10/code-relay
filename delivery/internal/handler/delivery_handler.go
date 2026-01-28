package handler

import (
	"delivery/internal/service"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// DeliveryHandler 处理 HTTP 请求
type DeliveryHandler struct {
	service *service.DeliveryService
}

// NewDeliveryHandler 创建新的 DeliveryHandler 实例
func NewDeliveryHandler(service *service.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{service: service}
}

// Response 统一响应格式
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// GetAccount 处理获取账号的请求
// POST /api/delivery/account
func (h *DeliveryHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed: only POST is supported")
		return
	}

	var req service.GetAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// 打印入参（token 做脱敏）
	skuVal := ""
	if req.SKU != nil {
		skuVal = *req.SKU
	}
	log.Printf("[delivery] GetAccount req: platform=%s product_code=%s sku=%s token=%s", req.Platform, req.ProductCode, skuVal, maskToken(req.Token))

	// 参数验证（返回具体缺失字段）
	missing := make([]string, 0, 3)
	if req.Token == "" {
		missing = append(missing, "token")
	}
	if req.ProductCode == "" {
		missing = append(missing, "product_code")
	}
	if req.Platform == "" {
		missing = append(missing, "platform")
	}
	if len(missing) > 0 {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")))
		return
	}

	// 调用 service 层
	resp, err := h.service.GetAccount(req)
	if err != nil {
		status, msg := mapServiceError(err)
		h.sendError(w, status, msg)
		return
	}

	h.sendSuccess(w, resp)
}

// sendSuccess 发送成功响应
func (h *DeliveryHandler) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// sendError 发送错误响应
func (h *DeliveryHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Code:    statusCode,
		Message: message,
	})
}

// mapServiceError 将 service 层错误映射为 HTTP 状态码与友好消息
func mapServiceError(err error) (int, string) {
	e := err.Error()
	switch {
	case strings.Contains(e, "invalid token"):
		return http.StatusUnauthorized, "invalid token"
	case strings.Contains(e, "invalid request body"):
		return http.StatusBadRequest, e
	case strings.Contains(e, "product not found"):
		return http.StatusNotFound, e
	case strings.Contains(e, "multiple products matched"):
		return http.StatusConflict, e
	case strings.Contains(e, "no available account"):
		return http.StatusNotFound, e
	default:
		return http.StatusInternalServerError, e
	}
}

// maskToken 对敏感 token 做脱敏打印（保留前2后2位）
func maskToken(s string) string {
	n := len(s)
	if n == 0 {
		return ""
	}
	if n <= 4 {
		return s[:1] + "***"
	}
	// 长度 > 4，保留前2后2位
	return s[:2] + "***" + s[n-2:]
}
