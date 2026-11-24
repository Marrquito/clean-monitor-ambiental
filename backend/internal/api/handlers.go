package api

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"clean-backend/internal/chat"
	"clean-backend/internal/sensors"
	"clean-backend/internal/storage"
)

const (
	maxMessageRunes  = 500
	maxMessageBytes  = 4 << 10
	defaultHistory   = 500
	maxHistoryLimit  = 5000
	maxHistoryWindow = 31 * 24 * time.Hour
	dbTimeout        = 5 * time.Second
	aiStatusTimeout  = 2 * time.Second
)

var bucketWidths = map[string]time.Duration{
	"1m":  time.Minute,
	"5m":  5 * time.Minute,
	"15m": 15 * time.Minute,
	"1h":  time.Hour,
}

var summaryPeriods = map[string]bool{
	chat.PeriodLastHour:    true,
	chat.PeriodLast2Hours:  true,
	chat.PeriodLast24Hours: true,
	chat.PeriodToday:       true,
}

type handler struct {
	deps Deps
	now  func() time.Time
}

func (h *handler) health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), aiStatusTimeout)
	defer cancel()

	database := "ok"
	if err := h.deps.Store.Ping(ctx); err != nil {
		slog.Warn("health: banco indisponível", "error", err)
		database = "error"
	}

	ollama := "ok"
	installed, err := h.deps.AI.ModelInstalled(ctx)
	if err != nil {
		ollama = "unavailable"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          healthStatus(database, ollama),
		"database":        database,
		"ollama":          ollama,
		"model":           h.deps.AI.Model(),
		"model_available": installed,
		"time":            h.now().UTC(),
	})
}

func healthStatus(database, ollama string) string {
	if database == "ok" && ollama == "ok" {
		return "ok"
	}
	return "degraded"
}

func (h *handler) latest(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	m, err := h.deps.Store.LatestMeasurement(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ainda não há medições registradas."})
		return
	}
	if err != nil {
		h.internalError(c, "consultar última medição", err)
		return
	}
	c.JSON(http.StatusOK, m)
}

type historyPoint struct {
	Time        time.Time `json:"time"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Light       float64   `json:"light"`
}

func (h *handler) history(c *gin.Context) {
	r, err := h.parseRange(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	if bucket := c.Query("bucket"); bucket != "" {
		width, ok := bucketWidths[bucket]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bucket deve ser 1m, 5m, 15m ou 1h"})
			return
		}
		buckets, err := h.deps.Store.Buckets(ctx, r, width)
		if err != nil {
			h.internalError(c, "consultar histórico agregado", err)
			return
		}
		points := make([]historyPoint, 0, len(buckets))
		for _, b := range buckets {
			points = append(points, historyPoint{
				Time:        b.Time,
				Temperature: round(b.Temperature, 2),
				Humidity:    round(b.Humidity, 2),
				Light:       round(b.Light, 1),
			})
		}
		c.JSON(http.StatusOK, gin.H{"bucket": bucket, "points": points})
		return
	}

	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	measurements, err := h.deps.Store.History(ctx, r, limit)
	if err != nil {
		h.internalError(c, "consultar histórico", err)
		return
	}
	points := make([]historyPoint, 0, len(measurements))
	for _, m := range measurements {
		points = append(points, historyPoint{
			Time:        m.MeasuredAt,
			Temperature: m.Temperature,
			Humidity:    m.Humidity,
			Light:       m.Light,
		})
	}
	c.JSON(http.StatusOK, gin.H{"bucket": nil, "points": points})
}

type periodJSON struct {
	Name string    `json:"name"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type aggregateJSON struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

type sourceJSON struct {
	Kind           string     `json:"kind"`
	Connected      bool       `json:"connected"`
	LastMeasuredAt *time.Time `json:"last_measured_at"`
}

type summaryResponse struct {
	Period      periodJSON           `json:"period"`
	Count       int                  `json:"count"`
	Current     *sensors.Measurement `json:"current"`
	Temperature *aggregateJSON       `json:"temperature"`
	Humidity    *aggregateJSON       `json:"humidity"`
	Source      sourceJSON           `json:"source"`
	GeneratedAt time.Time            `json:"generated_at"`
}

func (h *handler) summary(c *gin.Context) {
	name := c.DefaultQuery("period", chat.PeriodLast24Hours)
	if !summaryPeriods[name] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "period deve ser last_hour, last_2_hours, last_24_hours ou today"})
		return
	}
	now := h.now()
	period, err := chat.ResolvePeriod(name, "", "", now, h.deps.Location)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dbTimeout)
	defer cancel()

	stats, err := h.deps.Store.Stats(ctx, period.Range)
	if err != nil {
		h.internalError(c, "calcular resumo", err)
		return
	}
	resp := summaryResponse{
		Period:      periodJSON{Name: period.Name, From: period.Range.From, To: period.Range.To},
		Count:       stats.Count,
		Source:      sourceJSON{Kind: h.deps.Source.Kind(), Connected: h.deps.Source.Connected()},
		GeneratedAt: now.UTC(),
	}
	if stats.Count > 0 {
		resp.Temperature = &aggregateJSON{Min: round(stats.Temperature.Min, 1), Max: round(stats.Temperature.Max, 1), Avg: round(stats.Temperature.Avg, 1)}
		resp.Humidity = &aggregateJSON{Min: round(stats.Humidity.Min, 1), Max: round(stats.Humidity.Max, 1), Avg: round(stats.Humidity.Avg, 1)}
	}

	current, err := h.deps.Store.LatestMeasurement(ctx)
	switch {
	case err == nil:
		resp.Current = &current
	case !errors.Is(err, storage.ErrNotFound):
		h.internalError(c, "consultar leitura atual", err)
		return
	}

	last, found, err := h.deps.Store.LastMeasuredAt(ctx)
	if err != nil {
		h.internalError(c, "consultar última medição", err)
		return
	}
	if found {
		resp.Source.LastMeasuredAt = &last
	}
	c.JSON(http.StatusOK, resp)
}

type chatRequest struct {
	Message string `json:"message"`
}

type periodResponse struct {
	Name string    `json:"name"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (h *handler) chat(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxMessageBytes)
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "envie um JSON com o campo \"message\""})
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a mensagem não pode estar vazia"})
		return
	}
	if utf8.RuneCountInString(message) > maxMessageRunes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a mensagem deve ter no máximo 500 caracteres"})
		return
	}

	answer, err := h.deps.Assistant.Answer(c.Request.Context(), message)
	switch {
	case errors.Is(err, chat.ErrModelNotFound):
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "O modelo " + h.deps.AI.Model() + " não está instalado no Ollama. Execute: ollama pull " + h.deps.AI.Model(),
		})
		return
	case errors.Is(err, chat.ErrAIUnavailable):
		slog.Warn("chat: IA indisponível", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "A IA está indisponível no momento. Verifique se o Ollama está em execução e tente novamente.",
		})
		return
	case err != nil:
		h.internalError(c, "responder pergunta", err)
		return
	}

	var period *periodResponse
	if answer.Period != nil {
		period = &periodResponse{Name: answer.Period.Name, From: answer.Period.Range.From, To: answer.Period.Range.To}
	}
	c.JSON(http.StatusOK, gin.H{
		"answer": answer.Text,
		"model":  h.deps.AI.Model(),
		"intent": answer.Intent,
		"period": period,
	})
}

func (h *handler) parseRange(fromParam, toParam string) (storage.Range, error) {
	to := h.now().UTC()
	if toParam != "" {
		t, err := time.Parse(time.RFC3339, toParam)
		if err != nil {
			return storage.Range{}, errors.New("to deve estar no formato RFC3339")
		}
		to = t.UTC()
	}
	from := to.Add(-24 * time.Hour)
	if fromParam != "" {
		t, err := time.Parse(time.RFC3339, fromParam)
		if err != nil {
			return storage.Range{}, errors.New("from deve estar no formato RFC3339")
		}
		from = t.UTC()
	}
	if !from.Before(to) {
		return storage.Range{}, errors.New("from deve ser anterior a to")
	}
	if to.Sub(from) > maxHistoryWindow {
		return storage.Range{}, errors.New("o intervalo máximo de consulta é de 31 dias")
	}
	return storage.Range{From: from, To: to}, nil
}

func parseLimit(s string) (int, error) {
	if s == "" {
		return defaultHistory, nil
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("limit deve ser um inteiro positivo")
		}
		n = n*10 + int(r-'0')
		if n > maxHistoryLimit {
			return 0, errors.New("limit deve ser no máximo 5000")
		}
	}
	if n == 0 {
		return 0, errors.New("limit deve ser um inteiro positivo")
	}
	return n, nil
}

func (h *handler) internalError(c *gin.Context, action string, err error) {
	slog.Error("erro ao "+action, "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno ao " + action + "."})
}

func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
