package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	OpsBEBaseURL string
	OpsAPIKey    string
}

func Load() *Config {
	return &Config{
		OpsBEBaseURL: strings.TrimRight(os.Getenv("OPS_BE_BASE_URL"), "/"),
		OpsAPIKey:    os.Getenv("OPS_API_KEY"),
	}
}

func (c *Config) Validate() error {
	if c.OpsBEBaseURL == "" {
		return errors.New("OPS_BE_BASE_URL is required")
	}
	if c.OpsAPIKey == "" {
		return errors.New("OPS_API_KEY is required")
	}
	return nil
}
