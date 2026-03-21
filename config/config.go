package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func LoadFromEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func Load() *Config {
	LoadFromEnvFile(".env")

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
