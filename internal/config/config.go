package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type AgentConfig struct {
	Address        string
	PollInterval   int
	ReportInterval int
	Key            string
	RateLimit      int
	CryptoKey      string
}

type ServerConfig struct {
	Address         string
	LogLevel        string
	StoreInterval   int
	FileStoragePath string
	RestoreFlag     bool
	DatabaseDSN     string
	Key             string
	CryptoKey       string
	AuditFile       string
	AuditURL        string
}

func NewAgentConfig() (*AgentConfig, error) {
	v := viper.New()

	fs := pflag.NewFlagSet("agent", pflag.ContinueOnError)
	fs.StringP("address", "a", "localhost:8080", "server endpoint address")
	fs.IntP("poll-interval", "p", 2, "poll interval in seconds")
	fs.IntP("report-interval", "r", 10, "report interval in seconds")
	fs.StringP("key", "k", "", "hash key")
	fs.IntP("rate-limit", "l", 1, "rate limit for outgoing requests")
	fs.String("crypto-key", "", "path to public key file")
	fs.StringP("config", "c", "", "path to JSON config file")
	if err := fs.Parse(pflagArgs()); err != nil {
		return nil, fmt.Errorf("parse agent flags: %w", err)
	}

	if err := v.BindPFlags(fs); err != nil {
		return nil, err
	}

	// Env vars override.
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	// CONFIG env for config file path.
	cfgPath := v.GetString("config")
	if v.IsSet("CONFIG") {
		cfgPath = v.GetString("CONFIG")
	}
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("read agent config file: %w", err)
		}
		// Re-bind flags so they override file values.
		if err := v.BindPFlags(fs); err != nil {
			return nil, err
		}
	}

	// Handle duration fields from JSON (e.g. "1s" -> seconds int).
	pollInterval := v.GetInt("poll-interval")
	if pollInterval == 0 {
		if s := v.GetString("poll_interval"); s != "" {
			pollInterval = parseDurationSecondsOrDefault(s, 2)
		}
	}
	reportInterval := v.GetInt("report-interval")
	if reportInterval == 0 {
		if s := v.GetString("report_interval"); s != "" {
			reportInterval = parseDurationSecondsOrDefault(s, 10)
		}
	}

	addr := v.GetString("address")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		addr = "http://" + addr
	}

	return &AgentConfig{
		Address:        addr,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		Key:            v.GetString("key"),
		RateLimit:      v.GetInt("rate-limit"),
		CryptoKey:      v.GetString("crypto-key"),
	}, nil
}

func NewServerConfig() (*ServerConfig, error) {
	v := viper.New()

	fs := pflag.NewFlagSet("server", pflag.ContinueOnError)
	fs.StringP("address", "a", "localhost:8080", "server listen address")
	fs.StringP("log-level", "l", "info", "server log level")
	fs.IntP("store-interval", "i", 300, "store interval in seconds")
	fs.StringP("file-storage-path", "f", "", "file storage path")
	fs.BoolP("restore", "r", true, "restore data from file on startup")
	fs.StringP("database-dsn", "d", "", "database dsn")
	fs.StringP("key", "k", "", "hash key")
	fs.String("crypto-key", "", "path to private key file")
	fs.String("audit-file", "", "audit log file path")
	fs.String("audit-url", "", "audit log remote URL")
	fs.StringP("config", "c", "", "path to JSON config file")
	if err := fs.Parse(pflagArgs()); err != nil {
		return nil, fmt.Errorf("parse server flags: %w", err)
	}

	if err := v.BindPFlags(fs); err != nil {
		return nil, err
	}

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	cfgPath := v.GetString("config")
	if v.IsSet("CONFIG") {
		cfgPath = v.GetString("CONFIG")
	}
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
		if err := v.MergeInConfig(); err != nil {
			return nil, fmt.Errorf("read server config file: %w", err)
		}
		if err := v.BindPFlags(fs); err != nil {
			return nil, err
		}
	}

	// Handle duration field from JSON.
	storeInterval := v.GetInt("store-interval")
	if storeInterval == 0 {
		if s := v.GetString("store_interval"); s != "" {
			storeInterval = parseDurationSecondsOrDefault(s, 300)
		}
	}

	// JSON uses "store_file", flag uses "file-storage-path".
	fileStoragePath := v.GetString("file-storage-path")
	if fileStoragePath == "" {
		fileStoragePath = v.GetString("store_file")
	}

	// JSON uses "database_dsn", flag uses "database-dsn".
	databaseDSN := v.GetString("database-dsn")
	if databaseDSN == "" {
		databaseDSN = v.GetString("database_dsn")
	}

	// JSON uses "crypto_key", flag uses "crypto-key".
	cryptoKey := v.GetString("crypto-key")
	if cryptoKey == "" {
		cryptoKey = v.GetString("crypto_key")
	}

	return &ServerConfig{
		Address:         v.GetString("address"),
		LogLevel:        v.GetString("log-level"),
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
		RestoreFlag:     v.GetBool("restore"),
		DatabaseDSN:     databaseDSN,
		Key:             v.GetString("key"),
		CryptoKey:       cryptoKey,
		AuditFile:       v.GetString("audit-file"),
		AuditURL:        v.GetString("audit-url"),
	}, nil
}

func parseDurationSecondsOrDefault(s string, def int) int {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return int(d.Seconds())
}

// pflagArgs returns os.Args[1:] for pflag parsing.
func pflagArgs() []string {
	if len(os.Args) > 1 {
		return os.Args[1:]
	}
	return nil
}
