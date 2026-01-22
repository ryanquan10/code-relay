package client

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

type codexRelay struct{
	tokenConvertService *service.TokenConvertService
	proxy               *httputil.ReverseProxy
	upstreamURL         *url.URL
	mu                  sync.RWMutex
}

func NewCodexRelay() *codexRelay {
	return &codexRelay{
		tokenConvertService: service.NewTokenConvertService(),
	}
}

type contextKey string

const customerTokenContextKey contextKey = "customerToken"

// 每个用户一个独立的计数器
var (
	customerCounters = make(map[string]*service.TrafficCounter)
	countersMutex    sync.RWMutex
)

// getOrCreateCounter 获取或创建用户的流量计数器
func getOrCreateCounter(customerToken string) *service.TrafficCounter {
	countersMutex.RLock()
	counter, exists := customerCounters[customerToken]
	countersMutex.RUnlock()

	if exists {
		return counter
	}

	// 创建新计数器
	countersMutex.Lock()
	defer countersMutex.Unlock()

	// 双重检查
	if counter, exists := customerCounters[customerToken]; exists {
		return counter
	}

	counter = service.NewTrafficCounter(customerToken)
	customerCounters[customerToken] = counter
	log.Printf("[计数器] 为用户 %s 创建新的流量计数器", maskKey(customerToken))
	return counter
}

// maskKey 隐藏 key 的中间部分
func maskKey(key string) string {
	if len(key) <= 20 {
		return "***"
	}
	return key[:15] + "..." + key[len(key)-8:]
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

	// 解析上游 URL
	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		log.Fatalf("无法解析上游地址 %s: %v", upstream, err)
	}

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
	if proxyURL != "" {
		// 使用命令行指定的代理
		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			log.Fatalf("无法解析代理地址 %s: %v", proxyURL, err)
		}
		transport.Proxy = http.ProxyURL(proxyURLParsed)
		log.Printf("使用指定代理: %s", proxyURL)
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
		ctx := context.WithValue(req.Context(), customerTokenContextKey, customerToken)
		*req = *req.WithContext(ctx)

		// 使用 token_convert_service 转换 token 和上流地址
		upstreamConfig, err := c.tokenConvertService.ConvertToken(customerToken)
		if err != nil {
			log.Printf("[TokenConvert] 转换失败: %v, 使用默认配置", err)
			// 如果转换失败，使用默认的 token (兜底方案)
			req.Header.Set("Authorization", "Bearer sk-ant-oat01-3pOZJw3eh_LRataPfuRKvSS2_7I99bUKdX3AfbRfPkoHx3RbzoYSEaAa2NC3pdyERGr-zZLyE5vSRA5UeVNPH9gopj2NYAA")
		} else {
			// 使用转换后的上流 token
			req.Header.Set("Authorization", "Bearer "+upstreamConfig.UpstreamToken)
			log.Printf("[TokenConvert] 使用上流 URL: %s", upstreamConfig.UpstreamURL)
			// 注意：这里可以根据 upstreamConfig.UpstreamURL 动态修改请求的目标地址
			// 但由于当前的反向代理设计是在启动时固定上游地址，如果需要动态路由，
			// 需要重构代理逻辑，使用多个上游或者动态创建代理
		}
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", strings.Split(clientIP, ":")[0])

		// 日志记录
		log.Printf("转发请求: %s %s -> %s://%s%s [用户: %s]",
			req.Method, clientIP, upstreamURL.Scheme, upstreamURL.Host, req.URL.Path, maskKey(customerToken))

		if logHeaders {
			log.Printf("  请求头: %v", req.Header)
		}

		// 如果是流式请求，记录
		if strings.Contains(req.Header.Get("Accept"), "text/event-stream") {
			log.Printf("  检测到 SSE 流式请求")
		}

		// 统计请求流量 (上行)
		if req.Body != nil && req.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateCounter(customerToken)
			req.Body = service.NewCountingReadCloser(req.Body, counter)
		}
	}

	// 自定义响应修改
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 从 context 获取 customerToken
		customerToken := ""
		if key := resp.Request.Context().Value(customerTokenContextKey); key != nil {
			customerToken = key.(string)
		}

		// 统计响应流量 (下行)
		if resp.Body != nil && resp.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateCounter(customerToken)
			resp.Body = service.NewCountingReadCloser(resp.Body, counter)
		}

		log.Printf("收到响应: %s %d %s [用户: %s]",
			resp.Request.URL.Path, resp.StatusCode, resp.Status, maskKey(customerToken))

		if logHeaders {
			log.Printf("  响应头: %v", resp.Header)
		}

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("代理错误 [%s %s]: %v", r.Method, r.URL.Path, err)

		// 即使出错也尝试刷新计数器
		if key := r.Context().Value(customerTokenContextKey); key != nil {
			customerToken := key.(string)
			if counter := getOrCreateCounter(customerToken); counter != nil {
				if flushErr := counter.Flush(); flushErr != nil {
					log.Printf("[警告] 刷新计数器失败: %v", flushErr)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "代理转发失败", "details": "%v", "upstream": "%s"}`,
			err, upstream)
	}

	// 添加健康检查端点
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK - Relay to %s\n", upstream)
	})
	mux.Handle("/", proxy)

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
		countersMutex.RLock()
		for key, counter := range customerCounters {
			if err := counter.Flush(); err != nil {
				log.Printf("[警告] 刷新用户 %s 的计数器失败: %v", maskKey(key), err)
			} else {
				log.Printf("[完成] 用户 %s 的计数器已刷新", maskKey(key))
			}
		}
		countersMutex.RUnlock()

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
	// 默认上游地址
	upstreamURLStr := "https://code.newcli.com"

	// 解析上游 URL
	upstreamURL, err := url.Parse(upstreamURLStr)
	if err != nil {
		log.Printf("[Codex] 无法解析上游地址 %s: %v", upstreamURLStr, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)

	// 配置传输层
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 0,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// 自动检测系统代理
	transport.Proxy = http.ProxyFromEnvironment
	proxy.Transport = transport

	// 自定义请求修改
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		clientIP := req.RemoteAddr

		// 调用原始的 Director
		originalDirector(req)

		// 设置正确的 Host 头
		req.Host = upstreamURL.Host

		// 获取客户端 token
		customerToken := req.Header.Get("Authorization")
		if strings.HasPrefix(customerToken, "Bearer ") {
			customerToken = strings.TrimPrefix(customerToken, "Bearer ")
		}

		// 将 customerToken 存入 context
		ctx := context.WithValue(req.Context(), customerTokenContextKey, customerToken)
		*req = *req.WithContext(ctx)

		// 使用 token_convert_service 转换 token 和上流地址
		upstreamConfig, err := c.tokenConvertService.ConvertToken(customerToken)
		if err != nil {
			log.Printf("[Codex TokenConvert] 转换失败: %v, 使用默认配置", err)
			// 如果转换失败，使用默认的 token (兜底方案)
			req.Header.Set("Authorization", "Bearer sk-ant-oat01-3pOZJw3eh_LRataPfuRKvSS2_7I99bUKdX3AfbRfPkoHx3RbzoYSEaAa2NC3pdyERGr-zZLyE5vSRA5UeVNPH9gopj2NYAA")
		} else {
			// 使用转换后的上流 token
			req.Header.Set("Authorization", "Bearer "+upstreamConfig.UpstreamToken)
			log.Printf("[Codex TokenConvert] 使用上流 URL: %s", upstreamConfig.UpstreamURL)
		}

		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", strings.Split(clientIP, ":")[0])

		log.Printf("[Codex] 转发请求: %s %s -> %s://%s%s [用户: %s]",
			req.Method, clientIP, upstreamURL.Scheme, upstreamURL.Host, req.URL.Path, maskKey(customerToken))

		// 统计请求流量 (上行)
		if req.Body != nil && req.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateCounter(customerToken)
			req.Body = service.NewCountingReadCloser(req.Body, counter)
		}
	}

	// 自定义响应修改
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 从 context 获取 customerToken
		customerToken := ""
		if key := resp.Request.Context().Value(customerTokenContextKey); key != nil {
			customerToken = key.(string)
		}

		// 统计响应流量 (下行)
		if resp.Body != nil && resp.Body != http.NoBody && customerToken != "" {
			counter := getOrCreateCounter(customerToken)
			resp.Body = service.NewCountingReadCloser(resp.Body, counter)
		}

		log.Printf("[Codex] 收到响应: %s %d %s [用户: %s]",
			resp.Request.URL.Path, resp.StatusCode, resp.Status, maskKey(customerToken))

		return nil
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("[Codex] 代理错误 [%s %s]: %v", r.Method, r.URL.Path, err)

		// 即使出错也尝试刷新计数器
		if key := r.Context().Value(customerTokenContextKey); key != nil {
			customerToken := key.(string)
			if counter := getOrCreateCounter(customerToken); counter != nil {
				if flushErr := counter.Flush(); flushErr != nil {
					log.Printf("[Codex 警告] 刷新计数器失败: %v", flushErr)
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error": "代理转发失败", "details": "%v", "upstream": "%s"}`,
			err, upstreamURLStr)
	}

	// 执行代理
	proxy.ServeHTTP(w, r)
}
