package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 主配置文件结构（简化版，只包含需要的部分）
type Config struct {
	NSLookup NSLookupConfig `yaml:"nslookup"`
}

type NSLookupConfig struct {
	CheckInterval int      `yaml:"check_interval"`
	RemoteURLs    []string `yaml:"remote_urls"`
	RemoteURL     string   `yaml:"remote_url"` // 兼容旧配置
	AuthToken     string   `yaml:"auth_token"`
}

// IPUpdateRequest IP 更新请求
type IPUpdateRequest struct {
	IP        string `json:"ip"`
	Timestamp int64  `json:"timestamp"`
}

var (
	configPath string
	currentIP  string
)

func main() {
	flag.StringVar(&configPath, "config", "../config/app.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 兼容旧字段 remote_url
	if len(cfg.NSLookup.RemoteURLs) == 0 && cfg.NSLookup.RemoteURL != "" {
		cfg.NSLookup.RemoteURLs = []string{cfg.NSLookup.RemoteURL}
	}

	if len(cfg.NSLookup.RemoteURLs) == 0 {
		log.Fatalf("❌ 配置错误: remote_urls 不能为空")
	}

	log.Printf("🚀 本地 IP 监控服务启动")
	log.Printf("📡 检查间隔: %d 秒", cfg.NSLookup.CheckInterval)
	log.Printf("🔗 远程服务器: %s", strings.Join(cfg.NSLookup.RemoteURLs, ", "))

	// 初始检查
	if err := checkAndNotify(&cfg.NSLookup); err != nil {
		log.Printf("⚠️  初始检查失败: %v", err)
	}

	// 定时检查
	ticker := time.NewTicker(time.Duration(cfg.NSLookup.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := checkAndNotify(&cfg.NSLookup); err != nil {
			log.Printf("⚠️  检查失败: %v", err)
		}
	}
}

// loadConfig 加载配置文件
func loadConfig(path string) (*Config, error) {
	// 处理相对路径
	if !filepath.IsAbs(path) {
		execPath, _ := os.Executable()
		execDir := filepath.Dir(execPath)
		path = filepath.Join(execDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if cfg.NSLookup.CheckInterval == 0 {
		cfg.NSLookup.CheckInterval = 60 // 默认 60 秒
	}

	return &cfg, nil
}

// getPublicIP 获取公网 IP
func getPublicIP() (string, error) {
	// 使用 curl ifconfig.me
	cmd := exec.Command("curl", "-s", "--max-time", "10", "ifconfig.me")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("curl 执行失败: %w", err)
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", fmt.Errorf("获取到空 IP")
	}

	return ip, nil
}

// checkAndNotify 检查 IP 变化并通知远程服务器
func checkAndNotify(cfg *NSLookupConfig) error {
	// 获取当前公网 IP
	ip, err := getPublicIP()
	if err != nil {
		return fmt.Errorf("获取公网 IP 失败: %w", err)
	}

	log.Printf("📍 当前公网 IP: %s", ip)

	// 检查是否变化
	if ip == currentIP {
		log.Printf("✅ IP 未变化，无需通知")
		return nil
	}

	// IP 发生变化，通知远程服务器
	log.Printf("🔄 IP 发生变化: %s -> %s", currentIP, ip)
	currentIP = ip

	if err := notifyRemote(cfg, ip); err != nil {
		return fmt.Errorf("通知远程服务器失败: %w", err)
	}

	log.Printf("✅ 已通知远程服务器")
	return nil
}

// notifyRemote 通知远程服务器（对所有 URL）
func notifyRemote(cfg *NSLookupConfig, ip string) error {
	// 构建请求体
	req := IPUpdateRequest{
		IP:        ip,
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var errs []string

	for _, u := range cfg.RemoteURLs {
		if strings.TrimSpace(u) == "" {
			continue
		}

		httpReq, err := http.NewRequest("POST", u, bytes.NewReader(data))
		if err != nil {
			errs = append(errs, fmt.Sprintf("创建请求失败(%s): %v", u, err))
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+cfg.AuthToken)

		resp, err := client.Do(httpReq)
		if err != nil {
			errs = append(errs, fmt.Sprintf("发送请求失败(%s): %v", u, err))
			continue
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				errs = append(errs, fmt.Sprintf("服务器错误(%s): %d - %s", u, resp.StatusCode, string(body)))
			}
		}()
	}

	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "; "))
	}

	return nil
}
