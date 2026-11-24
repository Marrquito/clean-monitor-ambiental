package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrAIUnavailable = errors.New("IA indisponível")
	ErrModelNotFound = errors.New("modelo não encontrado no Ollama")
)

type OllamaClient struct {
	baseURL    string
	model      string
	keepAlive  string
	think      bool
	numPredict int
	http       *http.Client
}

type OllamaConfig struct {
	BaseURL    string
	Model      string
	Timeout    time.Duration
	KeepAlive  string
	Think      bool
	NumPredict int
}

func NewOllamaClient(cfg OllamaConfig) *OllamaClient {
	return &OllamaClient{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		keepAlive:  cfg.KeepAlive,
		think:      cfg.Think,
		numPredict: cfg.NumPredict,
		http:       &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *OllamaClient) Model() string {
	return c.model
}

type ChatInput struct {
	System      string
	User        string
	Format      json.RawMessage
	Temperature float64
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type chatRequest struct {
	Model     string          `json:"model"`
	Messages  []chatMessage   `json:"messages"`
	Stream    bool            `json:"stream"`
	Think     bool            `json:"think"`
	KeepAlive string          `json:"keep_alive"`
	Format    json.RawMessage `json:"format,omitempty"`
	Options   chatOptions     `json:"options"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
}

func (c *OllamaClient) Chat(ctx context.Context, in ChatInput) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: in.System},
			{Role: "user", Content: in.User},
		},
		Stream:    false,
		Think:     c.think,
		KeepAlive: c.keepAlive,
		Format:    in.Format,
		Options:   chatOptions{Temperature: in.Temperature, NumPredict: c.numPredict},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return "", fmt.Errorf("%w: %s", ErrModelNotFound, c.model)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("%w: status %d: %s", ErrAIUnavailable, resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var out chatResponse
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", fmt.Errorf("%w: resposta inválida: %v", ErrAIUnavailable, err)
	}
	return strings.TrimSpace(out.Message.Content), nil
}

type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// ModelInstalled reports whether the configured model is present in the Ollama server.
func (c *OllamaClient) ModelInstalled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrAIUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: status %d", ErrAIUnavailable, resp.StatusCode)
	}

	var tags tagsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tags); err != nil {
		return false, fmt.Errorf("%w: resposta inválida: %v", ErrAIUnavailable, err)
	}
	for _, m := range tags.Models {
		if m.Name == c.model || m.Name == c.model+":latest" {
			return true, nil
		}
	}
	return false, nil
}
