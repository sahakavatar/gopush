package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds configuration values
type Config struct {
	Redis struct {
		Nodes []struct {
			Address  string `json:"address"`
			Password string `json:"password"` // Password for each Redis node
		} `json:"nodes"`
		ChannelsPattern string `json:"channels_pattern"`
	} `json:"redis"`

	Server struct {
		Host      string `json:"host"`
		Port      string `json:"port"`
		Protocol  string `json:"protocol"`
		WsUrl     string `json:"ws_url"`
		Authorize struct {
			Url         string `json:"url"`
			Protocol    string `json:"protocol"`
			CashTimeOut int16  `json:"cash_time_out"`
		} `json:"authorize"`
		HealthCheckUrl string `json:"health_check_url"`
		TLS            struct {
			Enabled  bool   `json:"enabled"`
			CertFile string `json:"cert_file"`
			KeyFile  string `json:"key_file"`
		} `json:"tls"`
	} `json:"server"`

	Logging struct {
		Level string `json:"level"`
		File  string `json:"file"`
	} `json:"logging"`

	Environment string `json:"environment"`
}

// LoadConfig reads the configuration from a file
func LoadConfig(filePath string) (*Config, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file '%s': %v", filePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("failed to decode JSON config from '%s': %v", filePath, err)
	}

	// Validate required fields
	if len(config.Redis.Nodes) == 0 || config.Server.Host == "" || config.Server.Port == "" {
		return nil, fmt.Errorf("missing required configuration fields in '%s'", filePath)
	}

	return config, nil
}
