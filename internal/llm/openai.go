package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	apiKey       string
	baseURL      string
	model        string
	client       *http.Client
	maxDiffChars int
}

func NewClient(apiKey, baseURL, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		client:  &http.Client{},
	}
}

func NewClientWithMaxDiff(apiKey, baseURL, model string, maxDiffChars int) *Client {
	return &Client{
		apiKey:       apiKey,
		baseURL:      strings.TrimSuffix(baseURL, "/"),
		model:        model,
		client:       &http.Client{},
		maxDiffChars: maxDiffChars,
	}
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message chatMessage `json:"message"`
}

const defaultCommitPromptTemplate = `You are an expert at generating concise, conventional git commit messages.

Given the following git diff, generate a concise commit message (max 72 characters) that describes the changes.
Respond with ONLY the commit message, no quotes, no explanation.

%s

Commit message:`

var commitPromptTemplate = defaultCommitPromptTemplate

func SetPromptTemplate(tmpl string) {
	commitPromptTemplate = tmpl
}

func GetDefaultPromptTemplate() string {
	return defaultCommitPromptTemplate
}

func (c *Client) GenerateCommitMessage(diff string) (string, error) {
	if diff == "" {
		return "", fmt.Errorf("no diff provided")
	}

	prompt := fmt.Sprintf(commitPromptTemplate, diff)

	if c.maxDiffChars > 0 && len(diff) > c.maxDiffChars {
		prompt = fmt.Sprintf(commitPromptTemplate, diff[:c.maxDiffChars]+"\n... [diff truncated]")
	} else {
		prompt = fmt.Sprintf(commitPromptTemplate, diff)
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	message := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	return message, nil
}
