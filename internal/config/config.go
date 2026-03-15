package config

import (
	"flag"
	"log"
	"strings"

	"github.com/caarlos0/env/v6"
)

type AgentConfig struct {
	Address        string `env:"ADDRESS"`
	PollInterval   int    `env:"POLL_INTERVAL"`
	ReportInterval int    `env:"REPORT_INTERVAL"`
}

type ServerConfig struct {
	Address  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
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
	flag.Parse()

	cfg := ServerConfig{
		Address:  *addressFlag,
		LogLevel: *logLevel,
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

	return &cfg
}
