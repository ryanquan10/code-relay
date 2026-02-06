package handler

import (
	"codex-relay/pkg/utils"
	"delivery/internal/service"
	"encoding/json"
	"fmt"
	"io"
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

// GetAccountProductCode 处理通过内部产品码发货的请求
// GetAccountProductCode 处理通过内部产品码发货的请求
// GET|POST|PUT /api/delivery/product
func (h *DeliveryHandler) GetAccountProductCode(w http.ResponseWriter, r *http.Request) {
	// —— 无论方法为何，先打印请求行/头/Query/Body ——
	// 职责清晰：LogRequestDetails 只负责日志，不干涉业务参数解析。
	utils.LogRequestDetails(r) // ← 只需这一行即可完成所有调试日志

	// 允许 GET / POST / PUT，其他方法返回 405
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut:
		// ok
	default:
		h.sendError(w, http.StatusMethodNotAllowed, "method not allowed: only GET/POST/PUT are supported")
		return
	}

	// 使用与 service 对齐的请求结构
	var req service.DeliverProductRequest
	if r.Method == http.MethodGet {
		// GET 从 Query 取参
		q := r.URL.Query()
		req.Token = q.Get("token")
		req.ProductInnerCode = q.Get("product_inner_code")
		req.Platform = q.Get("platform")
	} else {
		// POST/PUT 优先解析 JSON Body；若无 Body 则回退到 Query
		bodyBytes, _ := io.ReadAll(r.Body)
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				h.sendError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
				return
			}
		} else {
			q := r.URL.Query()
			req.Token = q.Get("token")
			req.ProductInnerCode = q.Get("product_inner_code")
			req.Platform = q.Get("platform")
		}
	}

	if reqJSON, err := json.Marshal(req); err == nil {
		log.Printf("[delivery][debug] req-json: %s", string(reqJSON))
	} else {
		log.Printf("[delivery][debug] req-json marshal error: %v", err)
	}

	// 业务日志（按要求不脱敏）
	log.Printf("[delivery] GetAccountProductCode req: platform=%s product_inner_code=%s token=%s", req.Platform, req.ProductInnerCode, req.Token)

	missing := make([]string, 0, 3)
	if strings.TrimSpace(req.Token) == "" {
		missing = append(missing, "token")
	}
	if strings.TrimSpace(req.ProductInnerCode) == "" {
		missing = append(missing, "product_inner_code")
	}
	if strings.TrimSpace(req.Platform) == "" {
		missing = append(missing, "platform")
	}
	if len(missing) > 0 {
		h.sendError(w, http.StatusBadRequest, fmt.Sprintf("missing required fields: %s", strings.Join(missing, ", ")))
		return
	}

	// 调用 service 层
	data, err := h.service.DeliverProduct(req)
	if err != nil {
		status, msg := mapServiceError(err)
		h.sendError(w, status, msg)
		return
	}

	// 统一响应格式，data 为字符串
	h.sendSuccess(w, data)
}

// sendSuccess 发送成功响应
func (h *DeliveryHandler) sendSuccess(w http.ResponseWriter, data interface{}) {
	resp := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}

	// 打印最终返回的消息体
	if respJSON, err := json.Marshal(resp); err == nil {
		log.Printf("[delivery] response body: %s", string(respJSON))
	} else {
		log.Printf("[delivery] response body marshal error: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// sendError 发送错误响应
func (h *DeliveryHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	resp := Response{
		Code:    statusCode,
		Message: message,
	}

	// 打印最终返回的消息体
	if respJSON, err := json.Marshal(resp); err == nil {
		log.Printf("[delivery] response body: %s", string(respJSON))
	} else {
		log.Printf("[delivery] response body marshal error: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
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

/**

  1. codex (ID: 1)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "codex",
      "platform": "xianyu"
    }'

  2. xianyu-windsurf (ID: 2)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "xianyu-windsurf",
      "platform": "xianyu"
    }'

  3. codex-1d (ID: 4)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "codex-1d",
      "platform": "xianyu"
    }'

  4. claude-1d-infiniti (ID: 5)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "claude-1d-infiniti",
      "platform": "xianyu"
    }'

  5. codex-1m-infinite (ID: 6)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "codex-1m-infinite",
      "platform": "xianyu"
    }'

  6. claude-200dl (ID: 11)
  curl -X POST http://192.168.3.176:8089/api/delivery/product \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_inner_code": "claude-200dl",
      "platform": "xianyu"
    }'

  ---
  API 端点 2: /api/delivery/account (GetAccount)

  使用外部平台产品码 product_code

  各产品的 Curl 命令：

  codex (平台码: 1015498583373)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "codex",
      "platform": "xianyu"
    }'

  xianyu-windsurf (平台码: 1008138169938)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "xianyu-windsurf",
      "platform": "xianyu"
    }'

  codex-1d (平台码: 1015587199425)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "codex-1d",
      "platform": "xianyu"
    }'

  claude-1d-infiniti (平台码: 1017544089380)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "claude-1d-infiniti",
      "platform": "xianyu"
    }'

  codex-1m-infinite (平台码: 1017741022129)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "codex-1m-infinite",
      "platform": "xianyu"
    }'

  claude-200dl (平台码: 1015931191459)
  curl -X POST http://192.168.3.176:8089/api/delivery/account \
    -H "Content-Type: application/json" \
    -d '{
      "token": "secret_token11234567",
      "product_code": "claude-200dl",
      "platform": "xianyu"
    }'
*/
