package config

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/caarlos0/env/v6"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
}

type ServerConfig struct {
	Address         string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	RestoreFlag     bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

func NewAgentConfig() *AgentConfig {
	addressFlag := flag.String("a", "localhost:8080", "server endpoint address")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	reportInterval := flag.Int("r", 10, "report interval in seconds")
	flag.Parse()

	cfg := AgentConfig{
		Address:        *addressFlag,
		PollInterval:   *pollInterval,
		ReportInterval: *reportInterval,
	}

	var envCfg AgentConfig
	if err := env.Parse(&envCfg); err != nil {
		log.Fatal(err)
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

	if !strings.HasPrefix(cfg.Address, "http://") && !strings.HasPrefix(cfg.Address, "https://") {
		cfg.Address = "http://" + cfg.Address
	}

	return &cfg
}

func NewServerConfig() *ServerConfig {
	addressFlag := flag.String("a", "localhost:8080", "server listen address")
	logLevel := flag.String("l", "info", "server log level")
	storeInterval := flag.Int("i", 300, "store interval in seconds (0 for synchronous writes)")
	fileStoragePath := flag.String("f", "/tmp/metrics-db.json", "file storage path")
	restoreFlag := flag.Bool("r", true, "restore data from file on startup")
	databaseDSN := flag.String("d", "", "database dsn")
	flag.Parse()

	cfg := ServerConfig{
		Address:         *addressFlag,
		LogLevel:        *logLevel,
		StoreInterval:   *storeInterval,
		FileStoragePath: *fileStoragePath,
		RestoreFlag:     *restoreFlag,
		DatabaseDSN:     *databaseDSN,
	}

	var envCfg ServerConfig
	if err := env.Parse(&envCfg); err != nil {
		log.Fatal(err)
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
	if envCfg.FileStoragePath != "" {
		cfg.FileStoragePath = envCfg.FileStoragePath
	}
	// For boolean env variable, we need to check if it was explicitly set
	if os.Getenv("RESTORE") != "" {
		cfg.RestoreFlag = envCfg.RestoreFlag
	}

	return &cfg
}
