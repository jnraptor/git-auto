package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnvFile(t *testing.T) {
	t.Run("loads key-value pairs", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte("KEY1=bar\nKEY2=qux\n"), 0644); err != nil {
			t.Fatal(err)
		}

		if err := LoadFromEnvFile(envFile); err != nil {
			t.Fatalf("LoadFromEnvFile() error = %v", err)
		}

		defer os.Unsetenv("KEY1")
		defer os.Unsetenv("KEY2")

		if got := os.Getenv("KEY1"); got != "bar" {
			t.Errorf("KEY1 = %q, want %q", got, "bar")
		}
		if got := os.Getenv("KEY2"); got != "qux" {
			t.Errorf("KEY2 = %q, want %q", got, "qux")
		}
	})

	t.Run("skips empty lines and comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte("# comment\n\nKEY3=baz\n"), 0644); err != nil {
			t.Fatal(err)
		}

		LoadFromEnvFile(envFile)
		defer os.Unsetenv("KEY3")

		if got := os.Getenv("KEY3"); got != "baz" {
			t.Errorf("KEY3 = %q, want %q", got, "baz")
		}
	})

	t.Run("trims quotes from values", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte(`KEY4="quoted"`+"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		LoadFromEnvFile(envFile)
		defer os.Unsetenv("KEY4")

		if got := os.Getenv("KEY4"); got != "quoted" {
			t.Errorf("KEY4 = %q, want %q", got, "quoted")
		}
	})

	t.Run("skips malformed lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte("NOKEY\nKEY5=valid\n"), 0644); err != nil {
			t.Fatal(err)
		}

		LoadFromEnvFile(envFile)
		defer os.Unsetenv("KEY5")

		if got := os.Getenv("KEY5"); got != "valid" {
			t.Errorf("KEY5 = %q, want %q", got, "valid")
		}
	})

	t.Run("does not overwrite existing env vars", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte("KEY6=newvalue\n"), 0644); err != nil {
			t.Fatal(err)
		}

		os.Setenv("KEY6", "original")
		defer os.Unsetenv("KEY6")

		LoadFromEnvFile(envFile)

		if got := os.Getenv("KEY6"); got != "original" {
			t.Errorf("KEY6 = %q, want %q", got, "original")
		}
	})

	t.Run("returns error for nonexistent file", func(t *testing.T) {
		err := LoadFromEnvFile("/nonexistent/path/.env")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:  "key123",
				BaseURL: "https://api.openai.com/v1",
			},
			wantErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				APIKey:  "",
				BaseURL: "https://api.openai.com/v1",
			},
			wantErr: true,
		},
		{
			name: "missing base URL",
			config: Config{
				APIKey:  "key123",
				BaseURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "OPENAI_API_KEY",
		Message: "API key is required",
	}

	want := "config error: OPENAI_API_KEY - API key is required"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
