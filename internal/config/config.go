package config

import (
	"flag"
	"strings"
)

type AgentConfig struct {
	Address        string
	PollInterval   int
	ReportInterval int
}

type ServerConfig struct {
	Address string
}

func NewAgentConfig() *AgentConfig {

	addressFlag := flag.String("a", "http://localhost:8080", "server endpoint address")
	pollInterval := flag.Int("p", 2, "poll interval in seconds")
	reportInterval := flag.Int("r", 2, "report interval in seconds")
	flag.Parse()

	address := *addressFlag

	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	return &AgentConfig{
		Address:        address,
		PollInterval:   *pollInterval,
		ReportInterval: *reportInterval,
	}

}

func NewServerConfig() *ServerConfig {

	addressFlag := flag.String("a", "http://localhost:8080", "server endpoint address")
	flag.Parse()

	address := *addressFlag

	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}
	return &ServerConfig{
		Address: address,
	}

}
