package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	DB      DBConfig      `yaml:"db"`
	WeChat  WeChatConfig  `yaml:"wechat"`
	OpenAI  OpenAIConfig  `yaml:"openai"`
	Cron    CronConfig    `yaml:"cron"`
	Auth    AuthConfig    `yaml:"auth"`
	Notify  NotifyConfig  `yaml:"notify"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type DBConfig struct {
	Path string `yaml:"path"`
}

type WeChatConfig struct {
	AppID          string `yaml:"app_id"`
	AppSecret      string `yaml:"app_secret"`
	DefaultAuthor  string `yaml:"default_author"`
	DefaultThumbID string `yaml:"default_thumb_media_id"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

type CronConfig struct {
	DailyPublish string `yaml:"daily_publish"`
	StatusCheck  string `yaml:"status_check"`
}

type AuthConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type NotifyConfig struct {
	DingTalk DingTalkConfig `yaml:"dingtalk"`
	Email    EmailConfig    `yaml:"email"`
}

type DingTalkConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
	Secret     string `yaml:"secret"`
}

type EmailConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{Port: 8080, Host: "0.0.0.0"},
		DB:     DBConfig{Path: "data/wechat-autopub.db"},
		OpenAI: OpenAIConfig{Model: "gpt-4o"},
		Cron: CronConfig{
			DailyPublish: "0 8 * * *",
			StatusCheck:  "*/10 * * * *",
		},
		Auth: AuthConfig{Username: "admin", Password: "admin"},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	applyEnvOverrides(cfg)

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DB.Path = v
	}
	if v := os.Getenv("WECHAT_APP_ID"); v != "" {
		cfg.WeChat.AppID = v
	}
	if v := os.Getenv("WECHAT_APP_SECRET"); v != "" {
		cfg.WeChat.AppSecret = v
	}
	if v := os.Getenv("WECHAT_DEFAULT_AUTHOR"); v != "" {
		cfg.WeChat.DefaultAuthor = v
	}
	if v := os.Getenv("WECHAT_DEFAULT_THUMB_MEDIA_ID"); v != "" {
		cfg.WeChat.DefaultThumbID = v
	}
	if v := os.Getenv("OPENAI_API_KEY"); v != "" {
		cfg.OpenAI.APIKey = v
	}
	if v := os.Getenv("OPENAI_BASE_URL"); v != "" {
		cfg.OpenAI.BaseURL = v
	}
	if v := os.Getenv("OPENAI_MODEL"); v != "" {
		cfg.OpenAI.Model = v
	}
	if v := os.Getenv("AUTH_USERNAME"); v != "" {
		cfg.Auth.Username = v
	}
	if v := os.Getenv("AUTH_PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("DINGTALK_WEBHOOK_URL"); v != "" {
		cfg.Notify.DingTalk.WebhookURL = v
		cfg.Notify.DingTalk.Enabled = true
	}
	if v := os.Getenv("DINGTALK_SECRET"); v != "" {
		cfg.Notify.DingTalk.Secret = v
	}
}

func (c *Config) validate() error {
	if c.WeChat.AppID == "" {
		return fmt.Errorf("wechat.app_id is required")
	}
	if c.WeChat.AppSecret == "" {
		return fmt.Errorf("wechat.app_secret is required")
	}
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("openai.api_key is required")
	}
	return nil
}
