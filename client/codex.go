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

type codexRelay struct {
	tokenConvertService *service.TokenConvertService
	proxy               *httputil.ReverseProxy
	upstreamURL         *url.URL
	logHeaders          bool
	mu                  sync.RWMutex
}

// common.RelayConfig 代理配置
func NewCodexRelay() *codexRelay {
	return &codexRelay{
		tokenConvertService: service.NewTokenConvertService(),
	}
}

// NewCodexRelayWithConfig 使用配置创建 codex relay
func NewCodexRelayWithConfig(config common.RelayConfig) (*codexRelay, error) {
	relay := &codexRelay{
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

// 每个用户一个独立的计数器
// common.GetOrCreateCounter 获取或创建用户的流量计数器
// maskKey 隐藏 key 的中间部分
// setupProxy 配置反向代理（从 Relay 中提取的核心逻辑）
// 注意：此方法不加锁，调用者需要自行处理并发安全
func (c *codexRelay) setupProxy(config common.RelayConfig) error {
	// 解析上游 URL
	upstreamURL, err := url.Parse(config.UpstreamURL)
	if err != nil {
		return fmt.Errorf("无法解析上游地址 %s: %v", config.UpstreamURL, err)
	}
	c.upstreamURL = upstreamURL

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)

	// 配置传输层（支持 HTTPS 上游和代理）
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // 验证上游证书
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0, // 禁用响应头超时，支持流式响应
		ExpectContinueTimeout: 1 * time.Second,
	}

	// 配置代理
	if config.ProxyURL != "" {
		// 使用指定的代理
		proxyURLParsed, err := url.Parse(config.ProxyURL)
		if err != nil {
			return fmt.Errorf("无法解析代理地址 %s: %v", config.ProxyURL, err)
		}
		transport.Proxy = http.ProxyURL(proxyURLParsed)
		log.Printf("[Codex] 使用指定代理: %s", config.ProxyURL)
	} else {
		// 自动检测系统代理（HTTP_PROXY/HTTPS_PROXY 环境变量）
		transport.Proxy = http.ProxyFromEnvironment
	}

	proxy.Transport = transport

	// 自定义请求修改
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		// 记录原始请求
		clientIP := req.RemoteAddr

		// 调用原始的 Director
		originalDirector(req)

		// 设置正确的 Host 头
		req.Host = upstreamURL.Host
		customerToken := req.Header.Get("Authorization")

		// 去掉 "Bearer " 前缀
		if strings.HasPrefix(customerToken, "Bearer ") {
			customerToken = strings.TrimPrefix(customerToken, "Bearer ")
		}

		// 将 customerToken 存入 context
		ctx := context.WithValue(req.Context(), common.CustomerTokenContextKey, customerToken)
		var usageSession *service.CodexUsageSession
		if customerToken != "" {
			usageSession = service.NewCodexUsageSession(customerToken)
			ctx = context.WithValue(ctx, common.CodexUsageSessionContextKey, usageSession)
		}

		// 使用 token_convert_service 转换 token 和上流地址
		upstreamConfig, err := c.tokenConvertService.ConvertTokenAndCheck(customerToken)
		if err != nil {
			log.Printf("[Codex TokenConvert] 转换失败: %v, 使用默认配置", err)
			// 如果转换失败，使用默认的 token 测试用于
			req.Header.Set("Authorization", "Bearer "+customerToken)
			log.Printf("[Codex TokenConvert] 使用了缺省 token")
			log.Printf("[Codex TokenConvert] 失败时用户 customerToken: %s", common.MaskKey(customerToken))
		} else {
			// 将 upstreamConfig 存入 context（用于错误处理）
			ctx = context.WithValue(ctx, common.UpstreamConfigContextKey, upstreamConfig)

			// 使用转换后的上流 token
			req.Header.Set("Authorization", "Bearer "+upstreamConfig.UpstreamToken)
			log.Printf("[Codex TokenConvert] 使用上流 URL: %s", upstreamConfig.UpstreamURL)
			log.Printf("[Codex TokenConvert] 使用了用户 token")
			log.Printf("[Codex TokenConvert] 成功时 upstreamConfig: %+v", upstreamConfig)
		}

		*req = *req.WithContext(ctx)
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", strings.Split(clientIP, ":")[0])

		// 日志记录
		log.Printf("[Codex] 转发请求: %s %s -> %s://%s%s [用户: %s]",
			req.Method, clientIP, upstreamURL.Scheme, upstreamURL.Host, req.URL.Path, common.MaskKey(customerToken))

		if c.logHeaders {
			log.Printf("  请求头: %v", req.Header)
		}

		// 如果是流式请求，记录
		if strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
			log.Printf("  检测到 SSE 流式请求")
		}

		// 统计请求流量 (上行)
		if req.Body != nil && req.Body != http.NoBody && customerToken != "" {
			if usageSession != nil {
				req.Body = service.NewCodexCountingReadCloser(req.Body, usageSession, "in")
			} else {
				counter := common.GetOrCreateCounter(customerToken)
				req.Body = service.NewCountingReadCloser(req.Body, counter, "in")
			}
		}
	}

	// 自定义响应修改
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 从 context 获取 customerToken
		customerToken := ""
		if key := resp.Request.Context().Value(common.CustomerTokenContextKey); key != nil {
			customerToken = key.(string)
		}

		// 从 context 获取 upstreamConfig
		var upstreamConfig *service.UpstreamConfig
		if key := resp.Request.Context().Value(common.UpstreamConfigContextKey); key != nil {
			upstreamConfig = key.(*service.UpstreamConfig)
		}

		// 统计响应流量 (下行)
		if resp.Body != nil && resp.Body != http.NoBody && customerToken != "" {
			var usageSession *service.CodexUsageSession
			if v := resp.Request.Context().Value(common.CodexUsageSessionContextKey); v != nil {
				if s, ok := v.(*service.CodexUsageSession); ok {
					usageSession = s
				}
			}
			if usageSession != nil {
				resp.Body = service.NewCodexCountingReadCloser(resp.Body, usageSession, "out")
			} else {
				counter := common.GetOrCreateCounter(customerToken)
				resp.Body = service.NewCountingReadCloser(resp.Body, counter, "out")
			}
		}

		log.Printf("[Codex] 收到响应: %s %d %s [用户: %s]",
			resp.Request.URL.Path, resp.StatusCode, resp.Status, common.MaskKey(customerToken))

		if c.logHeaders {
			log.Printf("  响应头: %v", resp.Header)
		}

		// 检测上游错误（状态码 >= 400）
		if resp.StatusCode >= 400 && upstreamConfig != nil {
			requestPath := resp.Request.URL.Path
			errorMessage := resp.Status

			// 记录错误并降级上游源
			log.Printf("[Codex] 检测到上游错误: StatusCode=%d, SourceID=%d, Path=%s",
				resp.StatusCode, upstreamConfig.SourceID, requestPath)

			// 异步处理错误（避免阻塞响应）
			go c.tokenConvertService.HandleUpstreamError(
				upstreamConfig,
				requestPath,
				resp.StatusCode,
				errorMessage,
			)
		}

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[Codex] 代理错误 [%s %s]: %v", r.Method, r.URL.Path, err)

		// 即使出错也尝试写入用量（优先官方 usage，缺失则回退估算）
		if v := r.Context().Value(common.CodexUsageSessionContextKey); v != nil {
			if s, ok := v.(*service.CodexUsageSession); ok && s != nil {
				if flushErr := s.FinalizeAndSend(); flushErr != nil {
					log.Printf("[Codex 警告] 刷新用量失败: %v", flushErr)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "代理转发失败", "details": "%v", "upstream": "%s"}`,
			err, config.UpstreamURL)
	}

	c.proxy = proxy
	log.Printf("[Codex] 代理配置完成: 上游 %s", config.UpstreamURL)
	return nil
}

func (c *codexRelay) Relay() {
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

	flag.StringVar(&listen, "listen", ":8443", "本地监听地址，例如 :8443")
	flag.StringVar(&upstream, "upstream", "https://code.newcli.com", "上游地址（包含协议），例如 https://code.newcli.com")
	flag.StringVar(&proxyURL, "proxy", "", "代理地址，例如 http://127.0.0.1:7890（留空则使用环境变量或直连）")
	flag.BoolVar(&useHTTPS, "https", false, "是否使用 HTTPS 监听（需要证书）")
	flag.StringVar(&certFile, "cert", "server.crt", "HTTPS 证书文件路径")
	flag.StringVar(&keyFile, "key", "server.key", "HTTPS 私钥文件路径")
	flag.BoolVar(&logToFile, "log-file", false, "是否记录日志到文件")
	flag.StringVar(&logFilePath, "log-path", "http-relay.log", "日志文件路径")
	flag.BoolVar(&logHeaders, "log-headers", false, "是否记录 HTTP 请求头")
	flag.Parse()

	if listen == "" || upstream == "" {
		fmt.Println("用法：")
		fmt.Println("  http-relay.exe -listen :8443 -upstream https://code.newcli.com")
		fmt.Println("  http-relay.exe -listen :8443 -upstream https://code.newcli.com -proxy http://127.0.0.1:7890")
		fmt.Println("  http-relay.exe -listen :8443 -upstream https://code.newcli.com -https -cert server.crt -key server.key")
		os.Exit(1)
	}

	// 日志输出
	if logToFile {
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("无法打开日志文件: %v", err)
		}
		log.SetOutput(io.MultiWriter(os.Stdout, f))
		log.Printf("日志同时写入文件: %s", logFilePath)
	}

	// 使用 setupProxy 配置代理
	config := common.RelayConfig{
		UpstreamURL: upstream,
		ProxyURL:    proxyURL,
		LogHeaders:  logHeaders,
	}

	if err := c.setupProxy(config); err != nil {
		log.Fatalf("配置代理失败: %v", err)
	}

	// 添加健康检查端点
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK - Relay to %s\n", c.upstreamURL.String())
	})
	mux.Handle("/", c.proxy)

	// 设置 HTTP 服务器
	server := &http.Server{
		Addr:    listen,
		Handler: mux,
		// 流式响应需要禁用超时或设置很长的超时
		ReadTimeout:       0, // 禁用读超时
		WriteTimeout:      0, // 禁用写超时
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second, // 只限制读取头部的时间
	}

	protocol := "HTTP"
	if useHTTPS {
		protocol = "HTTPS"
	}
	log.Printf("%s 反向代理启动：监听 %s -> 上游 %s", protocol, listen, upstream)
	log.Printf("健康检查: %s://%s/health", strings.ToLower(protocol), listen)

	// 显示代理设置
	if proxyURL != "" {
		log.Printf("代理模式: 命令行指定 - %s", proxyURL)
	} else if envProxy := os.Getenv("HTTPS_PROXY"); envProxy != "" {
		log.Printf("代理模式: 环境变量 - %s", envProxy)
	} else if envProxy := os.Getenv("HTTP_PROXY"); envProxy != "" {
		log.Printf("代理模式: 环境变量 - %s", envProxy)
	} else {
		log.Printf("代理模式: 直连（未配置代理）")
	}

	// 优雅关闭
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("收到退出信号，正在刷新所有计数器...")

		// 刷新所有用户的计数器
		common.CountersMutex.RLock()
		for key, counter := range common.CustomerCounters {
			if err := counter.Flush(); err != nil {
				log.Printf("[警告] 刷新用户 %s 的计数器失败: %v", common.MaskKey(key), err)
			} else {
				log.Printf("[完成] 用户 %s 的计数器已刷新", common.MaskKey(key))
			}
		}
		common.CountersMutex.RUnlock()

		log.Println("正在关闭服务器...")
		_ = server.Close()
		os.Exit(0)
	}()

	// 启动服务器
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

var Codex = NewCodexRelay()

// GetAppType 返回客户端类型
func (c *codexRelay) GetAppType() string {
	return "codex"
}

func init() {
	// 自动注册到全局注册表
	Register(Codex)
}

// HandleRequest 处理HTTP请求并转发到上游
func (c *codexRelay) HandleRequest(w http.ResponseWriter, r *http.Request) {
	// 如果代理未配置，使用默认配置初始化
	if c.proxy == nil {
		log.Printf("[Codex] 代理未初始化，开始初始化...")
		c.mu.Lock()
		// 双重检查
		if c.proxy == nil {
			config := common.RelayConfig{
				UpstreamURL: "https://code.newcli.com",
				ProxyURL:    "", // 使用系统代理
				LogHeaders:  false,
			}
			log.Printf("[Codex] 正在配置代理，上游: %s", config.UpstreamURL)
			if err := c.setupProxy(config); err != nil {
				c.mu.Unlock()
				log.Printf("[Codex] 初始化代理失败: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			log.Printf("[Codex] 代理初始化完成")
		}
		c.mu.Unlock()
	}

	// 使用已配置的代理处理请求
	// 余额不足直接返回给下游客户端
	custTok := r.Header.Get("Authorization")
	if strings.HasPrefix(custTok, "Bearer ") {
		custTok = strings.TrimPrefix(custTok, "Bearer ")
	}
	if custTok != "" {
		if _, err := c.tokenConvertService.ConvertTokenAndCheck(custTok); err != nil {
			if strings.Contains(err.Error(), "insufficient balance") {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusPaymentRequired)
				fmt.Fprint(w, `{"error":"insufficient_balance","message":"账户余额不足，请充值"}`)
				return
			}
		}
	}

	log.Printf("[Codex] 开始处理请求: %s %s", r.Method, r.URL.Path)
	c.proxy.ServeHTTP(w, r)
}
