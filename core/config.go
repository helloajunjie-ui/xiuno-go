package core

import (
	"encoding/json"
	"os"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Cache    CacheConfig    `json:"cache"`
	JWT      JWTConfig      `json:"jwt"`
	Site     SiteConfig     `json:"site"`
	SMTP     []SMTPConfig   `json:"smtp,omitempty"` // SMTP 邮件发送配置
}

// SMTPConfig SMTP 服务器配置
type SMTPConfig struct {
	Email    string `json:"email"`    // 发件人邮箱
	Host     string `json:"host"`     // SMTP 服务器地址
	Port     int    `json:"port"`     // SMTP 端口（25/465/587）
	User     string `json:"user"`     // 用户名（通常为邮箱地址）
	Pass     string `json:"pass"`     // 密码或授权码
	Secure   string `json:"secure"`   // 加密方式: "tls" / "starttls" / ""（不加密）
	FromName string `json:"fromname"` // 发件人名称（可选）
}

type ServerConfig struct {
	Addr            string `json:"addr"`             // 监听地址，如 :8080
	ReadTimeout     int    `json:"read_timeout"`     // 秒
	WriteTimeout    int    `json:"write_timeout"`    // 秒
	ShutdownTimeout int    `json:"shutdown_timeout"` // 优雅停机超时秒数
}

type DatabaseConfig struct {
	DSN         string `json:"dsn"`          // user:pass@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True
	MaxOpen     int    `json:"max_open"`     // 最大打开连接数
	MaxIdle     int    `json:"max_idle"`     // 最大空闲连接数
	TablePrefix string `json:"table_prefix"` // 表前缀，默认 bbs_
}

type CacheConfig struct {
	Driver string      `json:"driver"` // "memory" 或 "redis"
	Redis  RedisConfig `json:"redis,omitempty"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type JWTConfig struct {
	Secret     string `json:"secret"`
	ExpireHour int    `json:"expire_hour"` // token 过期小时数
}

type SiteConfig struct {
	Name        string `json:"name"`
	Brief       string `json:"brief"`
	URL         string `json:"url"`
	CloseReason string `json:"close_reason,omitempty"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:            ":8080",
			ReadTimeout:     30,
			WriteTimeout:    30,
			ShutdownTimeout: 5,
		},
		Database: DatabaseConfig{
			DSN:         "root:root@tcp(127.0.0.1:3306)/xiuno?charset=utf8mb4&parseTime=True",
			MaxOpen:     20,
			MaxIdle:     5,
			TablePrefix: "bbs_",
		},
		Cache: CacheConfig{
			Driver: "memory",
		},
		JWT: JWTConfig{
			Secret:     "xiuno-go-secret-change-me",
			ExpireHour: 72,
		},
		Site: SiteConfig{
			Name:  "Xiuno Go",
			Brief: "Powered by Xiuno Go",
		},
		SMTP: []SMTPConfig{},
	}
}

// LoadConfig 从 JSON 文件加载配置
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil // 文件不存在则使用默认配置
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
