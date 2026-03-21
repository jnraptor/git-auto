package config

import (
	"os"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func Load() *Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	return &Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}
}

func (c *Config) Validate() error {
	if c.APIKey == "" {
		return &ConfigError{Field: "OPENAI_API_KEY", Message: "API key is required"}
	}
	if c.BaseURL == "" {
		return &ConfigError{Field: "OPENAI_BASE_URL", Message: "Base URL is required"}
	}
	return nil
}

type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " - " + e.Message
}
