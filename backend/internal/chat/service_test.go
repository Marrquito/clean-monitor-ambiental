package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"clean-backend/internal/sensors"
	"clean-backend/internal/storage"
)

type fakeStore struct {
	latest    sensors.Measurement
	latestErr error
	stats     storage.Stats
	extreme   storage.Extreme
	crossings storage.Crossings
}

func (f *fakeStore) LatestMeasurement(context.Context) (sensors.Measurement, error) {
	return f.latest, f.latestErr
}

func (f *fakeStore) Stats(context.Context, storage.Range) (storage.Stats, error) {
	return f.stats, nil
}

func (f *fakeStore) Extreme(_ context.Context, _ storage.Range, _ storage.Metric, highest bool) (storage.Extreme, error) {
	if f.extreme.MeasuredAt.IsZero() {
		return storage.Extreme{}, storage.ErrNotFound
	}
	return f.extreme, nil
}

func (f *fakeStore) Crossings(context.Context, storage.Range, storage.Metric, storage.Comparison, float64) (storage.Crossings, error) {
	return f.crossings, nil
}

// fakeModel answers the interpretation call with `interpretation` and the presentation call with a fixed sentence.
type fakeModel struct {
	interpretation string
	err            error
	calls          []ChatInput
}

func (m *fakeModel) Chat(_ context.Context, in ChatInput) (string, error) {
	m.calls = append(m.calls, in)
	if m.err != nil {
		return "", m.err
	}
	if len(in.Format) > 0 {
		return m.interpretation, nil
	}
	return "Resposta de teste.", nil
}

func (m *fakeModel) Model() string { return "qwen3.5:4b" }

func newTestService(store Store, model Model) *Service {
	s := NewService(store, model, sp)
	s.now = func() time.Time { return fixedNow }
	return s
}

func TestAnswerUnsupportedIntentReturnsClarificationWithoutDataAccess(t *testing.T) {
	model := &fakeModel{interpretation: `{"intent":"unsupported","period":"none"}`}
	svc := newTestService(&fakeStore{}, model)

	ans, err := svc.Answer(context.Background(), "Qual é a capital da França?")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if ans.Intent != IntentUnsupported || ans.Text != SupportedQueriesText() {
		t.Fatalf("resposta = %+v, esperado esclarecimento", ans)
	}
	if len(model.calls) != 1 {
		t.Fatalf("modelo chamado %d vezes, esperado apenas a interpretação", len(model.calls))
	}
}

func TestAnswerWithoutMeasurementsDoesNotInvent(t *testing.T) {
	model := &fakeModel{interpretation: `{"intent":"average_temperature","period":"last_hour"}`}
	svc := newTestService(&fakeStore{}, model)

	ans, err := svc.Answer(context.Background(), "Temperatura média na última hora?")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(ans.Text, "Não há medições registradas") {
		t.Fatalf("texto = %q, esperado aviso de ausência de dados", ans.Text)
	}
	if len(model.calls) != 1 {
		t.Fatalf("sem dados, o modelo não deve formular resposta; chamadas = %d", len(model.calls))
	}
}

func TestAnswerPresentsOnlyServerComputedFacts(t *testing.T) {
	now := fixedNow
	store := &fakeStore{
		stats:   storage.Stats{Count: 120, Temperature: storage.Aggregate{Avg: 24.25, Min: 20, Max: 31.7}},
		extreme: storage.Extreme{MeasuredAt: now.Add(-3 * time.Hour), Value: 31.7},
	}
	model := &fakeModel{interpretation: `{"intent":"max_temperature","period":"last_24_hours"}`}
	svc := newTestService(store, model)

	ans, err := svc.Answer(context.Background(), "Qual foi a temperatura máxima hoje?")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if ans.Text != "Resposta de teste." {
		t.Fatalf("texto = %q", ans.Text)
	}
	if ans.Period == nil || ans.Period.Name != PeriodLast24Hours {
		t.Fatalf("período ausente na resposta: %+v", ans.Period)
	}

	presentation := model.calls[len(model.calls)-1]
	if presentation.Format != nil {
		t.Fatal("apresentação não deve usar schema de interpretação")
	}
	var facts Facts
	payload := presentation.User[strings.Index(presentation.User, "Dados: ")+len("Dados: "):]
	if err := json.Unmarshal([]byte(payload), &facts); err != nil {
		t.Fatalf("fatos inválidos enviados ao modelo: %v", err)
	}
	if len(facts.Resultados) != 2 || facts.Resultados[0].Valor != "31,7 °C" {
		t.Fatalf("fatos = %+v, esperado máxima formatada em pt-BR", facts)
	}
}

func TestAnswerPropagatesAIUnavailable(t *testing.T) {
	model := &fakeModel{err: ErrAIUnavailable}
	svc := newTestService(&fakeStore{}, model)

	_, err := svc.Answer(context.Background(), "Como está o ambiente agora?")
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("erro = %v, esperado ErrAIUnavailable", err)
	}
}

func TestAnswerCurrentReadingsWithoutDataSaysSo(t *testing.T) {
	model := &fakeModel{interpretation: `{"intent":"current_readings","period":"none"}`}
	svc := newTestService(&fakeStore{latestErr: storage.ErrNotFound}, model)

	ans, err := svc.Answer(context.Background(), "Como está o ambiente agora?")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if ans.Period != nil || !strings.Contains(ans.Text, "Ainda não há medições") {
		t.Fatalf("resposta = %+v", ans)
	}
}

func TestAnswerThresholdWithoutCrossingsIsDeterministic(t *testing.T) {
	store := &fakeStore{stats: storage.Stats{Count: 50}, crossings: storage.Crossings{Count: 0}}
	model := &fakeModel{interpretation: `{"intent":"presence_or_threshold_query","metric":"humidity","operator":"above","threshold":80,"period":"today"}`}
	svc := newTestService(store, model)

	ans, err := svc.Answer(context.Background(), "Quando a umidade ultrapassou 80%?")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !strings.Contains(ans.Text, "Não houve medições de umidade acima de 80,0 %") {
		t.Fatalf("texto = %q", ans.Text)
	}
	if len(model.calls) != 1 {
		t.Fatalf("resultado vazio não deve acionar o modelo de apresentação")
	}
}
