package client

import "codex-relay/client/common"
import (
	"codex-relay/internal/service"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type geminiRelay struct {
	tokenConvertService *service.TokenConvertService
	proxy               *httputil.ReverseProxy
	upstreamURL         *url.URL
	logHeaders          bool
	mu                  sync.RWMutex
}

func NewGeminiRelay() *geminiRelay {
	return &geminiRelay{
		tokenConvertService: service.NewTokenConvertService(),
	}
}

// NewGeminiRelayWithConfig 使用配置创建 gemini relay
func NewGeminiRelayWithConfig(config common.RelayConfig) (*geminiRelay, error) {
	relay := &geminiRelay{
		tokenConvertService: service.NewTokenConvertService(),
		logHeaders:          config.LogHeaders,
	}

	// 如果提供了上游 URL，则初始化代理
	if config.UpstreamURL != "" {
		if err := relay.setupProxy(config); err != nil {
			return nil, err
		}
	}

	return relay, nil
}

// getOrCreateGeminiCounter 获取或创建用户的流量计数器（Gemini 独立映射）
var (
	geminiCustomerCounters = make(map[string]*service.GeminiTrafficCounter)
	geminiCountersMutex    sync.RWMutex
)

func getOrCreateGeminiCounter(customerToken string) *service.GeminiTrafficCounter {
	geminiCountersMutex.RLock()
	counter, exists := geminiCustomerCounters[customerToken]
	geminiCountersMutex.RUnlock()
	if exists {
		return counter
	}
	geminiCountersMutex.Lock()
	defer geminiCountersMutex.Unlock()
	if counter, exists := geminiCustomerCounters[customerToken]; exists {
		return counter
	}
	counter = service.NewGeminiTrafficCounter(customerToken)
	geminiCustomerCounters[customerToken] = counter
	log.Printf("[计数器/Gemini] 为用户 %s 创建新的流量计数器", common.MaskKey(customerToken))
	return counter
}

// setupProxy 配置反向代理（与 Codex 类似）
// 注意：此方法不加锁，调用者需要自行处理并发安全
func (c *geminiRelay) setupProxy(config common.RelayConfig) error {
	upstreamURL, err := url.Parse(config.UpstreamURL)
	if err != nil {
		return fmt.Errorf("无法解析上游地址 %s: %v", config.UpstreamURL, err)
	}
	c.upstreamURL = upstreamURL

	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: false},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if config.ProxyURL != "" {
		proxyURLParsed, err := url.Parse(config.ProxyURL)
		if err != nil {
			return fmt.Errorf("无法解析代理地址 %s: %v", config.ProxyURL, err)
		}
		transport.Proxy = http.ProxyURL(proxyURLParsed)
		log.Printf("[Gemini] 使用指定代理: %s", config.ProxyURL)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	proxy.Transport = transport

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		clientIP := req.RemoteAddr
		originalDirector(req)
		req.Host = upstreamURL.Host

		customerToken := req.Header.Get("Authorization")
		if strings.HasPrefix(customerToken, "Bearer ") {
			customerToken = strings.TrimPrefix(customerToken, "Bearer ")
		}

		ctx := context.WithValue(req.Context(), common.CustomerTokenContextKey, customerToken)

		upstreamConfig, err := c.tokenConvertService.ConvertTokenAndCheck(customerToken, req.URL.Path)
		if err != nil {
			log.Printf("[Gemini TokenConvert] 转换失败: %v, 使用默认配置", err)
			req.Header.Set("Authorization", "Bearer "+customerToken)
			log.Printf("[Gemini TokenConvert] 使用了缺省 token")
			log.Printf("[Gemini TokenConvert] 失败时用户 customerToken: %s", common.MaskKey(customerToken))
		} else {
			ctx = context.WithValue(ctx, common.UpstreamConfigContextKey, upstreamConfig)
			req.Header.Set("Authorization", "Bearer "+upstreamConfig.UpstreamToken)
			log.Printf("[Gemini TokenConvert] 使用上流 URL: %s", upstreamConfig.UpstreamURL)
			log.Printf("[Gemini TokenConvert] 使用了用户 token")
		}

		*req = *req.WithContext(ctx)
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", strings.Split(clientIP, ":")[0])

		log.Printf("[Gemini] 转发请求: %s %s -> %s://%s%s [用户: %s]", req.Method, clientIP, upstreamURL.Scheme, upstreamURL.Host, req.URL.Path, common.MaskKey(customerToken))
		if c.logHeaders {
			log.Printf("  请求头: %v", req.Header)
		}
		if strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
			log.Printf("  检测到 SSE 流式请求")
		}

		if req.Body != nil && req.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateGeminiCounter(customerToken)
			req.Body = service.NewGeminiCountingReadCloser(req.Body, counter, "in")
		}
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		customerToken := ""
		if key := resp.Request.Context().Value(common.CustomerTokenContextKey); key != nil {
			customerToken = key.(string)
		}
		var upstreamConfig *service.UpstreamConfig
		if key := resp.Request.Context().Value(common.UpstreamConfigContextKey); key != nil {
			upstreamConfig = key.(*service.UpstreamConfig)
		}

		if resp.Body != nil && resp.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateGeminiCounter(customerToken)
			resp.Body = service.NewGeminiCountingReadCloser(resp.Body, counter, "out")
		}

		log.Printf("[Gemini] 收到响应: %s %d %s [用户: %s]", resp.Request.URL.Path, resp.StatusCode, resp.Status, common.MaskKey(customerToken))
		if c.logHeaders {
			log.Printf("  响应头: %v", resp.Header)
		}

		if resp.StatusCode >= 400 && upstreamConfig != nil {
			requestPath := resp.Request.URL.Path
			errorMessage := resp.Status
			log.Printf("[Gemini] 检测到上游错误: StatusCode=%d, SourceID=%d, Path=%s", resp.StatusCode, upstreamConfig.SourceID, requestPath)
			go c.tokenConvertService.HandleUpstreamError(upstreamConfig, requestPath, resp.StatusCode, errorMessage)
		}
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[Gemini] 代理错误 [%s %s]: %v", r.Method, r.URL.Path, err)
		if key := r.Context().Value(common.CustomerTokenContextKey); key != nil {
			customerToken := key.(string)
			if counter := getOrCreateGeminiCounter(customerToken); counter != nil {
				if flushErr := counter.Flush(); flushErr != nil {
					log.Printf("[Gemini 警告] 刷新计数器失败: %v", flushErr)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "代理转发失败", "details": "%v", "upstream": "%s"}`, err, config.UpstreamURL)
	}

	c.proxy = proxy
	log.Printf("[Gemini] 代理配置完成: 上游 %s", config.UpstreamURL)
	return nil
}

func (c *geminiRelay) Relay() {
	var (
		listen      string
		upstream    string
		proxyURL    string
		useHTTPS    bool
		certFile    string
		keyFile     string
		logToFile   bool
		logFilePath string
		logHeaders  bool
	)

	flag.StringVar(&listen, "listen", ":8453", "本地监听地址，例如 :8453")
	flag.StringVar(&upstream, "upstream", "https://generativelanguage.googleapis.com/v1beta/openai", "上游地址（包含协议）")
	flag.StringVar(&proxyURL, "proxy", "", "代理地址，例如 http://127.0.0.1:7890（留空则使用环境变量或直连）")
	flag.BoolVar(&useHTTPS, "https", false, "是否使用 HTTPS 监听（需要证书）")
	flag.StringVar(&certFile, "cert", "server.crt", "HTTPS 证书文件路径")
	flag.StringVar(&keyFile, "key", "server.key", "HTTPS 私钥文件路径")
	flag.BoolVar(&logToFile, "log-file", false, "是否记录日志到文件")
	flag.StringVar(&logFilePath, "log-path", "http-relay-gemini.log", "日志文件路径")
	flag.BoolVar(&logHeaders, "log-headers", false, "是否记录 HTTP 请求头")
	flag.Parse()

	if listen == "" || upstream == "" {
		fmt.Println("用法：")
		fmt.Println("  http-relay.exe -listen :8453 -upstream https://generativelanguage.googleapis.com/v1beta/openai")
		fmt.Println("  http-relay.exe -listen :8453 -upstream https://generativelanguage.googleapis.com/v1beta/openai -proxy http://127.0.0.1:7890")
		fmt.Println("  http-relay.exe -listen :8453 -upstream https://generativelanguage.googleapis.com/v1beta/openai -https -cert server.crt -key server.key")
		os.Exit(1)
	}

	if logToFile {
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("无法打开日志文件: %v", err)
		}
		log.SetOutput(io.MultiWriter(os.Stdout, f))
		log.Printf("日志同时写入文件: %s", logFilePath)
	}

	config := common.RelayConfig{UpstreamURL: upstream, ProxyURL: proxyURL, LogHeaders: logHeaders}
	if err := c.setupProxy(config); err != nil {
		log.Fatalf("配置代理失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK - Relay to %s\n", c.upstreamURL.String())
	})
	mux.Handle("/", c.proxy)

	server := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadTimeout:       0,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	protocol := "HTTP"
	if useHTTPS {
		protocol = "HTTPS"
	}
	log.Printf("%s 反向代理启动：监听 %s -> 上游 %s", protocol, listen, upstream)
	log.Printf("健康检查: %s://%s/health", strings.ToLower(protocol), listen)

	if proxyURL != "" {
		log.Printf("代理模式: 命令行指定 - %s", proxyURL)
	} else if envProxy := os.Getenv("HTTPS_PROXY"); envProxy != "" {
		log.Printf("代理模式: 环境变量 - %s", envProxy)
	} else if envProxy := os.Getenv("HTTP_PROXY"); envProxy != "" {
		log.Printf("代理模式: 环境变量 - %s", envProxy)
	} else {
		log.Printf("代理模式: 直连（未配置代理）")
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("收到退出信号，正在刷新所有计数器...")
		geminiCountersMutex.RLock()
		for key, counter := range geminiCustomerCounters {
			if err := counter.Flush(); err != nil {
				log.Printf("[警告] 刷新用户 %s 的计数器失败: %v", common.MaskKey(key), err)
			} else {
				log.Printf("[完成] 用户 %s 的计数器已刷新", common.MaskKey(key))
			}
		}
		geminiCountersMutex.RUnlock()
		log.Println("正在关闭服务器...")
		_ = server.Close()
		os.Exit(0)
	}()

	var serverErr error
	if useHTTPS {
		serverErr = server.ListenAndServeTLS(certFile, keyFile)
	} else {
		serverErr = server.ListenAndServe()
	}
	if serverErr != nil && serverErr != http.ErrServerClosed {
		log.Fatalf("服务器错误: %v", serverErr)
	}
}

var Gemini = NewGeminiRelay()

func (c *geminiRelay) GetAppType() string { return "gemini" }

func init() {
	Register(Gemini)
}

// HandleRequest 处理HTTP请求并转发到上游
func (c *geminiRelay) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if c.proxy == nil {
		log.Printf("[Gemini] 代理未初始化，开始初始化...")
		c.mu.Lock()
		if c.proxy == nil {
			config := common.RelayConfig{UpstreamURL: "https://generativelanguage.googleapis.com/v1beta/openai", ProxyURL: "", LogHeaders: false}
			log.Printf("[Gemini] 正在配置代理，上游: %s", config.UpstreamURL)
			if err := c.setupProxy(config); err != nil {
				c.mu.Unlock()
				log.Printf("[Gemini] 初始化代理失败: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			log.Printf("[Gemini] 代理初始化完成")
		}
		c.mu.Unlock()
	}

	custTok := r.Header.Get("Authorization")
	if strings.HasPrefix(custTok, "Bearer ") {
		custTok = strings.TrimPrefix(custTok, "Bearer ")
	}
	if custTok != "" {
		if _, err := c.tokenConvertService.ConvertTokenAndCheck(custTok, r.URL.Path); err != nil {
			if strings.Contains(err.Error(), "insufficient balance") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusPaymentRequired)
				fmt.Fprint(w, `{"error":"insufficient_balance","message":"账户余额不足，请充值"}`)
				return
			}
			if strings.Contains(err.Error(), "account has expired") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"error":"account_expired","message":"账号已过期，无法继续使用"}`)
				return
			}
		}
	}

	log.Printf("[Gemini] 开始处理请求: %s %s", r.Method, r.URL.Path)
	c.proxy.ServeHTTP(w, r)
}
