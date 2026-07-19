package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
	Key            string `env:"KEY"`
	RateLimit      int    `env:"RATE_LIMIT"`
	CryptoKey      string `env:"CRYPTO_KEY"`
}

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	RestoreFlag     bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
}

// agentFileConfig is the JSON file config format for agent.
type agentFileConfig struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

// serverFileConfig is the JSON file config format for server.
type serverFileConfig struct {
	Address       string `json:"address"`
	Restore       *bool  `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
}

func NewAgentConfig() (*AgentConfig, error) {
	configFile := flag.String("c", "", "path to JSON config file")
	flag.StringVar(configFile, "config", "", "path to JSON config file")
	addressFlag := flag.String("a", "localhost:8080", "server endpoint address")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	keyFlag := flag.String("k", "", "hash key")
	rateLimitFlag := flag.Int("l", 1, "rate limit for outgoing requests")
	cryptoKeyFlag := flag.String("crypto-key", "", "path to public key file")
	flag.Parse()

	// Priority: flags (defaults) -> JSON file -> env vars override.
	cfg := AgentConfig{
		Address:        *addressFlag,
		PollInterval:   *pollInterval,
		ReportInterval: *reportInterval,
		Key:            *keyFlag,
		RateLimit:      *rateLimitFlag,
		CryptoKey:      *cryptoKeyFlag,
	}

	// CONFIG env overrides -c flag.
	cfgPath := *configFile
	if v := os.Getenv("CONFIG"); v != "" {
		cfgPath = v
	}

	// Apply JSON file config (lowest priority — only fill unset/default values).
	if cfgPath != "" {
		fileCfg, err := loadAgentFileConfig(cfgPath)
		if err != nil {
			return nil, err
		}
		if fileCfg.Address != "" && !isFlagSet("a") {
			cfg.Address = fileCfg.Address
		}
		if fileCfg.PollInterval != "" && !isFlagSet("p") {
			if secs, err := parseDurationSeconds(fileCfg.PollInterval); err == nil {
				cfg.PollInterval = secs
			}
		}
		if fileCfg.ReportInterval != "" && !isFlagSet("r") {
			if secs, err := parseDurationSeconds(fileCfg.ReportInterval); err == nil {
				cfg.ReportInterval = secs
			}
		}
		if fileCfg.CryptoKey != "" && !isFlagSet("crypto-key") {
			cfg.CryptoKey = fileCfg.CryptoKey
		}
	}

	// Env vars have highest priority.
	var envCfg AgentConfig
	if err := env.Parse(&envCfg); err != nil {
		return nil, fmt.Errorf("parse agent env config: %w", err)
	}
	if envCfg.Address != "" {
		cfg.Address = envCfg.Address
	}
	if envCfg.PollInterval != 0 {
		cfg.PollInterval = envCfg.PollInterval
	}
	if envCfg.ReportInterval != 0 {
		cfg.ReportInterval = envCfg.ReportInterval
	}
	if envCfg.Key != "" {
		cfg.Key = envCfg.Key
	}
	if envCfg.RateLimit != 0 {
		cfg.RateLimit = envCfg.RateLimit
	}
	if envCfg.CryptoKey != "" {
		cfg.CryptoKey = envCfg.CryptoKey
	}

	if !strings.HasPrefix(cfg.Address, "http://") && !strings.HasPrefix(cfg.Address, "https://") {
		cfg.Address = "http://" + cfg.Address
	}

	return &cfg, nil
}

func NewServerConfig() (*ServerConfig, error) {
	configFile := flag.String("c", "", "path to JSON config file")
	flag.StringVar(configFile, "config", "", "path to JSON config file")
	addressFlag := flag.String("a", "localhost:8080", "server listen address")
	logLevel := flag.String("l", "info", "server log level")
	storeInterval := flag.Int("i", 300, "store interval in seconds (0 for synchronous writes)")
	fileStoragePath := flag.String("f", "", "file storage path")
	restoreFlag := flag.Bool("r", true, "restore data from file on startup")
	databaseDSN := flag.String("d", "", "database dsn")
	keyFlag := flag.String("k", "", "hash key")
	cryptoKeyFlag := flag.String("crypto-key", "", "path to private key file")
	auditFileFlag := flag.String("audit-file", "", "audit log file path")
	auditURLFlag := flag.String("audit-url", "", "audit log remote URL")
	flag.Parse()

	cfg := ServerConfig{
		Address:         *addressFlag,
		LogLevel:        *logLevel,
		StoreInterval:   *storeInterval,
		FileStoragePath: *fileStoragePath,
		RestoreFlag:     *restoreFlag,
		DatabaseDSN:     *databaseDSN,
		Key:             *keyFlag,
		CryptoKey:       *cryptoKeyFlag,
		AuditFile:       *auditFileFlag,
		AuditURL:        *auditURLFlag,
	}

	cfgPath := *configFile
	if v := os.Getenv("CONFIG"); v != "" {
		cfgPath = v
	}

	if cfgPath != "" {
		fileCfg, err := loadServerFileConfig(cfgPath)
		if err != nil {
			return nil, err
		}
		if fileCfg.Address != "" && !isFlagSet("a") {
			cfg.Address = fileCfg.Address
		}
		if fileCfg.StoreInterval != "" && !isFlagSet("i") {
			if secs, err := parseDurationSeconds(fileCfg.StoreInterval); err == nil {
				cfg.StoreInterval = secs
			}
		}
		if fileCfg.StoreFile != "" && !isFlagSet("f") {
			cfg.FileStoragePath = fileCfg.StoreFile
		}
		if fileCfg.Restore != nil && !isFlagSet("r") {
			cfg.RestoreFlag = *fileCfg.Restore
		}
		if fileCfg.DatabaseDSN != "" && !isFlagSet("d") {
			cfg.DatabaseDSN = fileCfg.DatabaseDSN
		}
		if fileCfg.CryptoKey != "" && !isFlagSet("crypto-key") {
			cfg.CryptoKey = fileCfg.CryptoKey
		}
	}

	var envCfg ServerConfig
	if err := env.Parse(&envCfg); err != nil {
		return nil, fmt.Errorf("parse server env config: %w", err)
	}
	if envCfg.Address != "" {
		cfg.Address = envCfg.Address
	}
	if envCfg.LogLevel != "" {
		cfg.LogLevel = envCfg.LogLevel
	}
	if envCfg.StoreInterval != 0 {
		cfg.StoreInterval = envCfg.StoreInterval
	}
	if envCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = envCfg.DatabaseDSN
	}
	if envCfg.Key != "" {
		cfg.Key = envCfg.Key
	}
	if envCfg.CryptoKey != "" {
		cfg.CryptoKey = envCfg.CryptoKey
	}
	if envCfg.FileStoragePath != "" {
		cfg.FileStoragePath = envCfg.FileStoragePath
	}
	if os.Getenv("RESTORE") != "" {
		cfg.RestoreFlag = envCfg.RestoreFlag
	}
	if envCfg.AuditFile != "" {
		cfg.AuditFile = envCfg.AuditFile
	}
	if envCfg.AuditURL != "" {
		cfg.AuditURL = envCfg.AuditURL
	}

	return &cfg, nil
}

func loadAgentFileConfig(path string) (*agentFileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent config file: %w", err)
	}
	var fc agentFileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse agent config file: %w", err)
	}
	return &fc, nil
}

func loadServerFileConfig(path string) (*serverFileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read server config file: %w", err)
	}
	var fc serverFileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse server config file: %w", err)
	}
	return &fc, nil
}

// parseDurationSeconds parses a duration string (e.g. "1s", "10s", "300s") and returns seconds.
func parseDurationSeconds(s string) (int, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return int(d.Seconds()), nil
}

// isFlagSet returns true if the named flag was explicitly set on command line.
func isFlagSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
