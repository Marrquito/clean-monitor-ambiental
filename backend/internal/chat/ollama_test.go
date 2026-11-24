package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newOllamaForTest(t *testing.T, handler http.HandlerFunc) *OllamaClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewOllamaClient(OllamaConfig{
		BaseURL:    srv.URL,
		Model:      "qwen3.5:4b",
		Timeout:    2 * time.Second,
		KeepAlive:  "10m",
		Think:      false,
		NumPredict: 128,
	})
}

func TestOllamaChatSendsConfiguredPayload(t *testing.T) {
	var got map[string]any
	client := newOllamaForTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Errorf("rota inesperada: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"  olá  "}}`))
	})

	out, err := client.Chat(context.Background(), ChatInput{System: "sys", User: "usr", Format: InterpretationSchema(), Temperature: 0})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if out != "olá" {
		t.Fatalf("conteúdo = %q, esperado conteúdo sem espaços", out)
	}
	if got["model"] != "qwen3.5:4b" || got["stream"] != false || got["think"] != false || got["keep_alive"] != "10m" {
		t.Fatalf("payload incorreto: %v", got)
	}
	if _, ok := got["format"].(map[string]any); !ok {
		t.Fatal("schema de formato não foi enviado")
	}
}

func TestOllamaChatMapsErrors(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr error
	}{
		{"modelo ausente", http.StatusNotFound, ErrModelNotFound},
		{"erro do servidor", http.StatusInternalServerError, ErrAIUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newOllamaForTest(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":"x"}`))
			})
			_, err := client.Chat(context.Background(), ChatInput{User: "x"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("erro = %v, esperado %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaChatTimeoutIsUnavailable(t *testing.T) {
	client := newOllamaForTest(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	})
	client.http.Timeout = 100 * time.Millisecond

	_, err := client.Chat(context.Background(), ChatInput{User: "x"})
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("erro = %v, esperado ErrAIUnavailable após timeout", err)
	}
}

func TestOllamaConnectionRefusedIsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	client := NewOllamaClient(OllamaConfig{BaseURL: url, Model: "m", Timeout: time.Second})
	_, err := client.Chat(context.Background(), ChatInput{User: "x"})
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("erro = %v, esperado ErrAIUnavailable", err)
	}
}

func TestModelInstalledMatchesTag(t *testing.T) {
	client := newOllamaForTest(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3.5:4b"},{"name":"llama3:latest"}]}`))
	})
	ok, err := client.ModelInstalled(context.Background())
	if err != nil || !ok {
		t.Fatalf("ModelInstalled = %v, %v; esperado true", ok, err)
	}
}
