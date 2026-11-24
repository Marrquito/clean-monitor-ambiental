package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"clean-backend/internal/sensors"
	"clean-backend/internal/storage"
)

type Store interface {
	LatestMeasurement(ctx context.Context) (sensors.Measurement, error)
	Stats(ctx context.Context, r storage.Range) (storage.Stats, error)
	Extreme(ctx context.Context, r storage.Range, m storage.Metric, highest bool) (storage.Extreme, error)
	Crossings(ctx context.Context, r storage.Range, m storage.Metric, c storage.Comparison, threshold float64) (storage.Crossings, error)
}

type Model interface {
	Chat(ctx context.Context, in ChatInput) (string, error)
	Model() string
}

type Service struct {
	store Store
	model Model
	loc   *time.Location
	now   func() time.Time
}

func NewService(store Store, model Model, loc *time.Location) *Service {
	return &Service{store: store, model: model, loc: loc, now: time.Now}
}

type Answer struct {
	Text   string
	Intent string
	Period *Period
}

// Answer runs the controlled flow: the model only classifies the question and
// phrases the result; every number comes from a parameterized query.
func (s *Service) Answer(ctx context.Context, message string) (Answer, error) {
	raw, err := s.interpret(ctx, message)
	if errors.Is(err, ErrUnparseable) {
		return Answer{Text: SupportedQueriesText(), Intent: IntentUnsupported}, nil
	}
	if err != nil {
		return Answer{}, err
	}

	q, err := ValidateQuery(raw, s.now(), s.loc)
	if errors.Is(err, ErrUnsupportedIntent) {
		return Answer{Text: SupportedQueriesText(), Intent: IntentUnsupported}, nil
	}
	if err != nil {
		return Answer{Text: err.Error(), Intent: raw.Intent}, nil
	}

	res := Answer{Intent: q.Intent}
	if q.Period.Name != "" {
		p := q.Period
		res.Period = &p
	}

	facts, direct, err := s.collect(ctx, q)
	if err != nil {
		return Answer{}, err
	}
	if direct != "" {
		res.Text = direct
		return res, nil
	}

	text, err := s.present(ctx, message, facts)
	if err != nil {
		return Answer{}, err
	}
	res.Text = text
	return res, nil
}

func (s *Service) interpret(ctx context.Context, message string) (RawIntent, error) {
	content, err := s.model.Chat(ctx, ChatInput{
		System:      s.interpreterPrompt(),
		User:        message,
		Format:      InterpretationSchema(),
		Temperature: 0,
	})
	if err != nil {
		return RawIntent{}, err
	}
	var raw RawIntent
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return RawIntent{}, fmt.Errorf("%w: %v", ErrUnparseable, err)
	}
	return raw, nil
}

func (s *Service) present(ctx context.Context, message string, facts Facts) (string, error) {
	payload, err := json.Marshal(facts)
	if err != nil {
		return "", err
	}
	text, err := s.model.Chat(ctx, ChatInput{
		System:      presenterPrompt,
		User:        fmt.Sprintf("Pergunta: %s\nDados: %s", message, payload),
		Temperature: 0.2,
	})
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", fmt.Errorf("%w: resposta vazia", ErrAIUnavailable)
	}
	return text, nil
}

func (s *Service) interpreterPrompt() string {
	now := s.now().In(s.loc)
	return fmt.Sprintf(`Você classifica perguntas sobre medições de sensores ambientais (temperatura, umidade e luminosidade).
Responda somente com JSON no formato solicitado. Não responda à pergunta e não escreva SQL.
Data e hora atuais no fuso %s: %s.

Intenções permitidas:
- current_readings: %s
- average_temperature: %s
- max_temperature: %s
- min_temperature: %s
- average_humidity: %s
- max_humidity: %s
- temperature_trend: %s
- humidity_trend: %s
- historical_comparison: %s
- presence_or_threshold_query: %s
- unsupported: a pergunta não corresponde a nenhuma intenção acima.

Períodos: last_hour (última hora), last_2_hours (últimas 2 horas), last_24_hours (últimas 24 horas), today (hoje), yesterday (ontem), last_7_days (últimos 7 dias), this_week (esta semana, desde segunda-feira), last_week (semana passada), custom (somente quando a pergunta citar datas explícitas: preencha obrigatoriamente start_date e end_date no formato AAAA-MM-DD), none (use apenas com current_readings).
Se a pergunta não citar período, use last_24_hours.

Exemplos:
"Como está o ambiente agora?" -> {"intent":"current_readings","period":"none"}
"Qual foi a temperatura máxima hoje?" -> {"intent":"max_temperature","period":"today"}
"Qual foi a maior temperatura dessa semana?" -> {"intent":"max_temperature","period":"this_week"}
"Qual a umidade média dos últimos 7 dias?" -> {"intent":"average_humidity","period":"last_7_days"}
"Qual foi a temperatura média entre 20/11 e 22/11?" -> {"intent":"average_temperature","period":"custom","start_date":"%d-11-20","end_date":"%d-11-22"}
"Quando a umidade ultrapassou 80%%?" -> {"intent":"presence_or_threshold_query","metric":"humidity","operator":"above","threshold":80,"period":"last_24_hours"}
"Compare a temperatura da última hora com a anterior." -> {"intent":"historical_comparison","metric":"temperature","period":"last_hour"}`,
		s.loc, now.Format("2006-01-02 15:04 MST")+" ("+weekdaysPT[now.Weekday()]+")",
		intentDescriptions[IntentCurrentReadings],
		intentDescriptions[IntentAverageTemperature],
		intentDescriptions[IntentMaxTemperature],
		intentDescriptions[IntentMinTemperature],
		intentDescriptions[IntentAverageHumidity],
		intentDescriptions[IntentMaxHumidity],
		intentDescriptions[IntentTemperatureTrend],
		intentDescriptions[IntentHumidityTrend],
		intentDescriptions[IntentHistoricalCompare],
		intentDescriptions[IntentThresholdQuery],
		now.Year(), now.Year(),
	)
}

var weekdaysPT = [...]string{"domingo", "segunda-feira", "terça-feira", "quarta-feira", "quinta-feira", "sexta-feira", "sábado"}

const presenterPrompt = `Você é o assistente do CLEAN, um sistema de monitoramento ambiental.
Responda em português do Brasil, em no máximo três frases curtas, usando somente os dados do JSON fornecido.
Não faça cálculos novos, não invente valores e não mencione que é um modelo de linguagem.
Se os dados não forem suficientes para responder, diga isso.`
