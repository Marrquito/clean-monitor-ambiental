package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"clean-backend/internal/chat"
	"clean-backend/internal/sensors"
	"clean-backend/internal/storage"
)

type fakeStore struct {
	latestErr error
	pingErr   error
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }
func (f *fakeStore) LatestMeasurement(context.Context) (sensors.Measurement, error) {
	return sensors.Measurement{Temperature: 23.4, Source: "simulated"}, f.latestErr
}
func (f *fakeStore) LastMeasuredAt(context.Context) (time.Time, bool, error) {
	return time.Time{}, false, nil
}
func (f *fakeStore) History(context.Context, storage.Range, int) ([]sensors.Measurement, error) {
	return nil, nil
}
func (f *fakeStore) Buckets(context.Context, storage.Range, time.Duration) ([]storage.Bucket, error) {
	return nil, nil
}
func (f *fakeStore) Stats(context.Context, storage.Range) (storage.Stats, error) {
	return storage.Stats{}, nil
}

type fakeAssistant struct {
	ans chat.Answer
	err error
}

func (f *fakeAssistant) Answer(context.Context, string) (chat.Answer, error) {
	return f.ans, f.err
}

type fakeAI struct{}

func (fakeAI) ModelInstalled(context.Context) (bool, error) { return true, nil }
func (fakeAI) Model() string                                { return "qwen3.5:4b" }

func newTestRouter(store DataStore, assistant Assistant) http.Handler {
	loc, _ := time.LoadLocation("America/Sao_Paulo")
	return NewRouter(Deps{
		Store:     store,
		Assistant: assistant,
		AI:        fakeAI{},
		Source:    NoSource{},
		Location:  loc,
	})
}

func do(t *testing.T, r http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestHealthReportsDegradedWhenDatabaseDown(t *testing.T) {
	r := newTestRouter(&fakeStore{pingErr: errors.New("down")}, &fakeAssistant{})
	rec := do(t, r, http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["database"] != "error" || body["status"] != "degraded" {
		t.Fatalf("corpo = %v", body)
	}
}

func TestLatestReturns404WhenEmpty(t *testing.T) {
	r := newTestRouter(&fakeStore{latestErr: storage.ErrNotFound}, &fakeAssistant{})
	if rec := do(t, r, http.MethodGet, "/api/sensors/latest", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, esperado 404", rec.Code)
	}
}

func TestHistoryValidatesParameters(t *testing.T) {
	r := newTestRouter(&fakeStore{}, &fakeAssistant{})
	tests := []string{
		"/api/sensors/history?from=ontem",
		"/api/sensors/history?from=2026-10-27T10:00:00Z&to=2026-10-27T09:00:00Z",
		"/api/sensors/history?bucket=7m",
		"/api/sensors/history?limit=99999",
		"/api/sensors/history?limit=abc",
	}
	for _, target := range tests {
		if rec := do(t, r, http.MethodGet, target, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, esperado 400", target, rec.Code)
		}
	}
}

func TestChatRejectsInvalidMessages(t *testing.T) {
	r := newTestRouter(&fakeStore{}, &fakeAssistant{})
	tests := map[string]string{
		"vazia":         `{"message":"   "}`,
		"muito longa":   `{"message":"` + strings.Repeat("a", 501) + `"}`,
		"sem campo":     `{"texto":"oi"}`,
		"json inválido": `{"message":`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			if rec := do(t, r, http.MethodPost, "/api/chat", body); rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, esperado 400", rec.Code)
			}
		})
	}
}

func TestChatMapsAIUnavailableTo503(t *testing.T) {
	r := newTestRouter(&fakeStore{}, &fakeAssistant{err: chat.ErrAIUnavailable})
	rec := do(t, r, http.MethodPost, "/api/chat", `{"message":"como está o ambiente?"}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, esperado 503", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "ErrAIUnavailable") || !strings.Contains(rec.Body.String(), "indisponível") {
		t.Fatalf("mensagem deve ser amigável: %s", rec.Body.String())
	}
}

func TestChatReturnsAnswer(t *testing.T) {
	r := newTestRouter(&fakeStore{}, &fakeAssistant{ans: chat.Answer{Text: "Está 23,4 °C.", Intent: chat.IntentCurrentReadings}})
	rec := do(t, r, http.MethodPost, "/api/chat", `{"message":"como está o ambiente?"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Answer string `json:"answer"`
		Model  string `json:"model"`
		Intent string `json:"intent"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Answer != "Está 23,4 °C." || body.Model != "qwen3.5:4b" || body.Intent != chat.IntentCurrentReadings {
		t.Fatalf("corpo = %+v", body)
	}
}

func TestSummaryRejectsUnknownPeriod(t *testing.T) {
	r := newTestRouter(&fakeStore{}, &fakeAssistant{})
	if rec := do(t, r, http.MethodGet, "/api/dashboard/summary?period=last_week", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, esperado 400", rec.Code)
	}
}
