package handler

import (
	"delivery/internal/service"
	"encoding/json"
	"net/http"
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
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req service.GetAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// 参数验证
	if req.Token == "" || req.ProductCode == "" || req.Platform == "" {
		h.sendError(w, http.StatusBadRequest, "token, product_code and platform are required")
		return
	}

	// 调用 service 层
	resp, err := h.service.GetAccount(req)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, err.Error())
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
