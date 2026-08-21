package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

type ServerConfig struct {
	BindCtrl    string
	BindVisitor string
	BindAPI     string
	Token       string
	LogLevel    string
}

type ClientConfig struct {
	ServerAddr string
	LocalAddr  string
	Token      string
	ClientID   string
	BindHealth string
	LogLevel   string
	MaxIdle    int
	MaxActive  int
	IdleTTL    time.Duration
	PoolWait   time.Duration
}

type DemoConfig struct {
	BindHTTP string
	StaticDir string
	NodeID   string
	LogLevel string
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func overlayJSON(path string, dst any) error {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

func LoadServer() (ServerConfig, error) {
	cfg := ServerConfig{
		BindCtrl:    "0.0.0.0:7000",
		BindVisitor: "0.0.0.0:8080",
		BindAPI:     "0.0.0.0:9090",
		Token:       "simplefrp-dev-token",
		LogLevel:    "info",
	}
	if err := overlayJSON(os.Getenv("SIMPLEFRP_CONFIG"), &cfg); err != nil {
		return cfg, err
	}
	cfg.BindCtrl = getenv("SIMPLEFRP_BIND_CTRL", cfg.BindCtrl)
	cfg.BindVisitor = getenv("SIMPLEFRP_BIND_VISITOR", cfg.BindVisitor)
	cfg.BindAPI = getenv("SIMPLEFRP_BIND_API", cfg.BindAPI)
	cfg.Token = getenv("SIMPLEFRP_TOKEN", cfg.Token)
	cfg.LogLevel = getenv("SIMPLEFRP_LOG_LEVEL", cfg.LogLevel)
	if cfg.Token == "" {
		return cfg, fmt.Errorf("SIMPLEFRP_TOKEN is required")
	}
	return cfg, nil
}

func LoadClient() (ClientConfig, error) {
	cfg := ClientConfig{
		ServerAddr: "127.0.0.1:7000",
		LocalAddr:  "127.0.0.1:8080",
		Token:      "simplefrp-dev-token",
		ClientID:   "edge-01",
		BindHealth: "0.0.0.0:9091",
		LogLevel:   "info",
		MaxIdle:    8,
		MaxActive:  128,
		IdleTTL:    30 * time.Second,
		PoolWait:   5 * time.Second,
	}
	if err := overlayJSON(os.Getenv("SIMPLEFRP_CONFIG"), &cfg); err != nil {
		return cfg, err
	}
	cfg.ServerAddr = getenv("SIMPLEFRP_SERVER_ADDR", cfg.ServerAddr)
	cfg.LocalAddr = getenv("SIMPLEFRP_LOCAL_ADDR", cfg.LocalAddr)
	cfg.Token = getenv("SIMPLEFRP_TOKEN", cfg.Token)
	cfg.ClientID = getenv("SIMPLEFRP_CLIENT_ID", cfg.ClientID)
	cfg.BindHealth = getenv("SIMPLEFRP_BIND_HEALTH", cfg.BindHealth)
	cfg.LogLevel = getenv("SIMPLEFRP_LOG_LEVEL", cfg.LogLevel)
	cfg.MaxIdle = getenvInt("SIMPLEFRP_POOL_MAX_IDLE", cfg.MaxIdle)
	cfg.MaxActive = getenvInt("SIMPLEFRP_POOL_MAX_ACTIVE", cfg.MaxActive)
	cfg.IdleTTL = getenvDuration("SIMPLEFRP_POOL_IDLE_TTL", cfg.IdleTTL)
	cfg.PoolWait = getenvDuration("SIMPLEFRP_POOL_WAIT", cfg.PoolWait)
	if cfg.Token == "" || cfg.ServerAddr == "" || cfg.LocalAddr == "" {
		return cfg, fmt.Errorf("token, server addr and local addr are required")
	}
	return cfg, nil
}

func LoadDemo() DemoConfig {
	return DemoConfig{
		BindHTTP:  getenv("SIMPLEFRP_BIND_HTTP", "0.0.0.0:8080"),
		StaticDir: getenv("SIMPLEFRP_STATIC_DIR", "/app/static"),
		NodeID:    getenv("SIMPLEFRP_NODE_ID", "intranet-alpha-7"),
		LogLevel:  getenv("SIMPLEFRP_LOG_LEVEL", "info"),
	}
}

func NextBackoff(prev time.Duration) time.Duration {
	if prev <= 0 {
		return 500 * time.Millisecond
	}
	next := prev * 2
	if next > 2*time.Second {
		return 2 * time.Second
	}
	return next
}
