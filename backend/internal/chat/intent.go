package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"clean-backend/internal/storage"
)

const (
	IntentCurrentReadings    = "current_readings"
	IntentAverageTemperature = "average_temperature"
	IntentMaxTemperature     = "max_temperature"
	IntentMinTemperature     = "min_temperature"
	IntentAverageHumidity    = "average_humidity"
	IntentMaxHumidity        = "max_humidity"
	IntentTemperatureTrend   = "temperature_trend"
	IntentHumidityTrend      = "humidity_trend"
	IntentHistoricalCompare  = "historical_comparison"
	IntentThresholdQuery     = "presence_or_threshold_query"
	IntentUnsupported        = "unsupported"

	periodNone    = "none"
	defaultPeriod = PeriodLast24Hours
)

var intentDescriptions = map[string]string{
	IntentCurrentReadings:    "leituras atuais de temperatura, umidade e luminosidade (sem período)",
	IntentAverageTemperature: "temperatura média no período",
	IntentMaxTemperature:     "temperatura máxima no período e horário em que ocorreu",
	IntentMinTemperature:     "temperatura mínima no período e horário em que ocorreu",
	IntentAverageHumidity:    "umidade média no período",
	IntentMaxHumidity:        "umidade máxima no período e horário em que ocorreu",
	IntentTemperatureTrend:   "se a temperatura está subindo, descendo ou estável no período",
	IntentHumidityTrend:      "se a umidade está subindo, descendo ou estável no período",
	IntentHistoricalCompare:  "comparação de média entre o período e o período anterior de mesma duração (metric: temperature, humidity ou light)",
	IntentThresholdQuery:     "quando e quantas vezes uma métrica ficou acima ou abaixo de um valor (metric, operator above|below, threshold numérico)",
}

var (
	ErrUnsupportedIntent = errors.New("intenção não suportada")
	ErrUnparseable       = errors.New("resposta de interpretação inválida")
)

// RawIntent is the structured interpretation the model returns. It is untrusted
// input and must pass ValidateQuery before any database access.
type RawIntent struct {
	Intent    string   `json:"intent"`
	Metric    string   `json:"metric"`
	Period    string   `json:"period"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	Operator  string   `json:"operator"`
	Threshold *float64 `json:"threshold"`
}

type Query struct {
	Intent     string
	Metric     storage.Metric
	Period     Period
	Comparison storage.Comparison
	Threshold  float64
}

func InterpretationSchema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "intent": {"type": "string", "enum": [
      "current_readings", "average_temperature", "max_temperature", "min_temperature",
      "average_humidity", "max_humidity", "temperature_trend", "humidity_trend",
      "historical_comparison", "presence_or_threshold_query", "unsupported"]},
    "metric": {"type": "string", "enum": ["temperature", "humidity", "light"]},
    "period": {"type": "string", "enum": [
      "last_hour", "last_2_hours", "last_24_hours", "today", "yesterday",
      "last_7_days", "this_week", "last_week", "custom", "none"]},
    "start_date": {"type": "string"},
    "end_date": {"type": "string"},
    "operator": {"type": "string", "enum": ["above", "below"]},
    "threshold": {"type": "number"}
  },
  "required": ["intent", "period"]
}`)
}

// ValidateQuery checks the model's interpretation against the allowed intents and
// periods, and returns an error message suitable for the user when it does not.
func ValidateQuery(raw RawIntent, now time.Time, loc *time.Location) (Query, error) {
	if _, ok := intentDescriptions[raw.Intent]; !ok {
		return Query{}, ErrUnsupportedIntent
	}

	q := Query{Intent: raw.Intent}
	if raw.Intent == IntentCurrentReadings {
		return q, nil
	}

	periodName := raw.Period
	if periodName == "" || periodName == periodNone {
		periodName = defaultPeriod
	}
	period, err := ResolvePeriod(periodName, raw.StartDate, raw.EndDate, now, loc)
	if err != nil {
		return Query{}, err
	}
	q.Period = period

	switch raw.Intent {
	case IntentHistoricalCompare:
		metric, err := parseMetric(raw.Metric, storage.MetricTemperature)
		if err != nil {
			return Query{}, err
		}
		q.Metric = metric
	case IntentThresholdQuery:
		metric, err := parseMetric(raw.Metric, "")
		if err != nil {
			return Query{}, err
		}
		if metric == "" {
			return Query{}, fmt.Errorf("informe se o limite se refere à temperatura, umidade ou luminosidade")
		}
		if raw.Threshold == nil {
			return Query{}, fmt.Errorf("informe o valor do limite que deseja consultar")
		}
		q.Metric = metric
		q.Threshold = *raw.Threshold
		q.Comparison = storage.Above
		if raw.Operator == "below" {
			q.Comparison = storage.Below
		}
	case IntentMaxHumidity, IntentAverageHumidity, IntentHumidityTrend:
		q.Metric = storage.MetricHumidity
	default:
		q.Metric = storage.MetricTemperature
	}
	return q, nil
}

func parseMetric(s string, def storage.Metric) (storage.Metric, error) {
	switch storage.Metric(s) {
	case storage.MetricTemperature, storage.MetricHumidity, storage.MetricLight:
		return storage.Metric(s), nil
	case "":
		return def, nil
	}
	return "", fmt.Errorf("métrica não suportada: %q", s)
}

func SupportedQueriesText() string {
	return "Posso responder sobre: leituras atuais, temperatura e umidade médias, máximas e mínimas, " +
		"tendência de temperatura ou umidade, comparação entre períodos e quando um limite foi ultrapassado. " +
		"Períodos aceitos: última hora, últimas 2 horas, últimas 24 horas, hoje, ontem, últimos 7 dias, esta semana, semana passada ou um intervalo de datas."
}
