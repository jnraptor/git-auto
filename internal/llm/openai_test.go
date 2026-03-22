package llm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test-key", "https://api.openai.com/v1", "gpt-4")

	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-key")
	}
	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.openai.com/v1")
	}
	if client.model != "gpt-4" {
		t.Errorf("model = %q, want %q", client.model, "gpt-4")
	}
	if client.client == nil {
		t.Error("client should not be nil")
	}
	if client.maxDiffChars != 0 {
		t.Errorf("maxDiffChars = %d, want 0", client.maxDiffChars)
	}
}

func TestNewClientWithMaxDiff(t *testing.T) {
	client := NewClientWithMaxDiff("test-key", "https://api.openai.com/v1", "gpt-4", 1000)

	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", client.apiKey, "test-key")
	}
	if client.maxDiffChars != 1000 {
		t.Errorf("maxDiffChars = %d, want 1000", client.maxDiffChars)
	}
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("key", "https://api.openai.com/v1/", "gpt-3.5-turbo")

	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.openai.com/v1")
	}
}

func TestSetPromptTemplate(t *testing.T) {
	original := commitPromptTemplate
	defer func() { commitPromptTemplate = original }()

	customPrompt := "Custom: %s"
	SetPromptTemplate(customPrompt)

	if commitPromptTemplate != customPrompt {
		t.Errorf("commitPromptTemplate = %q, want %q", commitPromptTemplate, customPrompt)
	}
}

func TestGetDefaultPromptTemplate(t *testing.T) {
	defaultTmpl := GetDefaultPromptTemplate()
	if defaultTmpl != defaultCommitPromptTemplate {
		t.Errorf("GetDefaultPromptTemplate() = %q, want %q", defaultTmpl, defaultCommitPromptTemplate)
	}
}

func TestGenerateCommitMessageWithMaxDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"test message"}}]}`))
	}))
	defer server.Close()

	client := NewClientWithMaxDiff("test-key", server.URL, "gpt-3.5-turbo", 10)

	longDiff := strings.Repeat("a", 100)
	_, err := client.GenerateCommitMessage(longDiff)
	if err != nil {
		t.Fatalf("GenerateCommitMessage() error = %v", err)
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	t.Run("returns error for empty diff", func(t *testing.T) {
		client := NewClient("test-key", "http://localhost", "gpt-3.5-turbo")
		_, err := client.GenerateCommitMessage("")
		if err == nil {
			t.Error("expected error for empty diff")
		}
		if err.Error() != "no diff provided" {
			t.Errorf("error = %q, want %q", err.Error(), "no diff provided")
		}
	})

	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer test-key")
			}
			if r.Method != http.MethodPost {
				t.Errorf("Method = %q, want %q", r.Method, http.MethodPost)
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices":[{"message":{"content":"fix: update config"}}]}`))
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, "gpt-3.5-turbo")
		msg, err := client.GenerateCommitMessage("+ added new feature")
		if err != nil {
			t.Fatalf("GenerateCommitMessage() error = %v", err)
		}
		if msg != "fix: update config" {
			t.Errorf("msg = %q, want %q", msg, "fix: update config")
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer server.Close()

		client := NewClient("invalid-key", server.URL, "gpt-3.5-turbo")
		_, err := client.GenerateCommitMessage("+ some change")
		if err == nil {
			t.Fatal("expected error for API failure")
		}
	})

	t.Run("empty choices response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices":[]}`))
		}))
		defer server.Close()

		client := NewClient("test-key", server.URL, "gpt-3.5-turbo")
		_, err := client.GenerateCommitMessage("+ some change")
		if err == nil {
			t.Fatal("expected error for empty choices")
		}
		if err.Error() != "no response choices returned" {
			t.Errorf("error = %q, want %q", err.Error(), "no response choices returned")
		}
	})

	t.Run("HTTP request failure", func(t *testing.T) {
		client := NewClient("test-key", "http://localhost:1", "gpt-3.5-turbo")
		_, err := client.GenerateCommitMessage("+ some change")
		if err == nil {
			t.Fatal("expected error for failed request")
		}
	})
}
