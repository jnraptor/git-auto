package llm

import (
	"net/http"
	"net/http/httptest"
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
}

func TestNewClientTrimsTrailingSlash(t *testing.T) {
	client := NewClient("key", "https://api.openai.com/v1/", "gpt-3.5-turbo")

	if client.baseURL != "https://api.openai.com/v1" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://api.openai.com/v1")
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
