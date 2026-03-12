package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr     string        `yaml:"listen_addr"`
	DataDir        string        `yaml:"data_dir"`
	HMACSecret     string        `yaml:"hmac_secret"`
	HMACMaxDrift   time.Duration `yaml:"hmac_max_drift"`
	MaxUploadSize  int64         `yaml:"max_upload_size"`
	LogLevel       string        `yaml:"log_level"`
	LogFormat      string        `yaml:"log_format"`
	LogFile        string        `yaml:"log_file"`
	TrustedProxies []string      `yaml:"trusted_proxies"`
}

func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr:    ":8080",
		DataDir:       "./data",
		HMACMaxDrift:  5 * time.Minute,
		MaxUploadSize: 536870912,
		LogLevel:      "info",
		LogFormat:     "json",
	}

	if path := os.Getenv("CONFIG_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parsing config file: %w", err)
		}
	}

	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("HMAC_SECRET"); v != "" {
		cfg.HMACSecret = v
	}
	if v := os.Getenv("HMAC_MAX_DRIFT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("parsing HMAC_MAX_DRIFT: %w", err)
		}
		cfg.HMACMaxDrift = d
	}
	if v := os.Getenv("MAX_UPLOAD_SIZE"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing MAX_UPLOAD_SIZE: %w", err)
		}
		cfg.MaxUploadSize = n
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}
	if v := os.Getenv("LOG_FILE"); v != "" {
		cfg.LogFile = v
	}
	if v := os.Getenv("TRUSTED_PROXIES"); v != "" {
		var proxies []string
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				proxies = append(proxies, p)
			}
		}
		cfg.TrustedProxies = proxies
	}

	if cfg.HMACSecret == "" {
		return nil, fmt.Errorf("hmac_secret is required")
	}
	if cfg.HMACMaxDrift <= 0 || cfg.HMACMaxDrift > 15*time.Minute {
		return nil, fmt.Errorf("hmac_max_drift must be between 0 and 15m (got %s)", cfg.HMACMaxDrift)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[cfg.LogLevel] {
		return nil, fmt.Errorf("log_level must be one of: debug, info, warn, error (got %q)", cfg.LogLevel)
	}

	validFormats := map[string]bool{"json": true, "text": true}
	if !validFormats[cfg.LogFormat] {
		return nil, fmt.Errorf("log_format must be one of: json, text (got %q)", cfg.LogFormat)
	}

	return cfg, nil
}
