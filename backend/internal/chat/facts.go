package chat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"clean-backend/internal/storage"
)

type Facts struct {
	Consulta   string   `json:"consulta"`
	Periodo    string   `json:"periodo,omitempty"`
	Amostras   int      `json:"amostras,omitempty"`
	Resultados []Result `json:"resultados"`
}

type Result struct {
	Rotulo string `json:"rotulo"`
	Valor  string `json:"valor"`
}

type metricInfo struct {
	name     string
	unit     string
	decimals int
	// stableRate is the per-hour change below which a trend is reported as stable.
	stableRate float64
}

var metrics = map[storage.Metric]metricInfo{
	storage.MetricTemperature: {name: "temperatura", unit: " °C", decimals: 1, stableRate: 0.2},
	storage.MetricHumidity:    {name: "umidade", unit: " %", decimals: 1, stableRate: 1.0},
	storage.MetricLight:       {name: "luminosidade", unit: " lux", decimals: 0, stableRate: 20},
}

func (m metricInfo) format(v float64) string {
	return fmtNumber(v, m.decimals) + m.unit
}

func (m metricInfo) formatSigned(v float64) string {
	return fmtSigned(v, m.decimals) + m.unit
}

// collect gathers the facts for a query. A non-empty direct answer means the
// result is fully determined by the data and no model call is needed.
func (s *Service) collect(ctx context.Context, q Query) (Facts, string, error) {
	switch q.Intent {
	case IntentCurrentReadings:
		return s.current(ctx)
	case IntentHistoricalCompare:
		return s.compare(ctx, q)
	case IntentThresholdQuery:
		return s.threshold(ctx, q)
	case IntentMaxTemperature, IntentMinTemperature, IntentMaxHumidity:
		return s.extreme(ctx, q)
	default:
		return s.aggregate(ctx, q)
	}
}

func (s *Service) current(ctx context.Context) (Facts, string, error) {
	m, err := s.store.LatestMeasurement(ctx)
	if errors.Is(err, storage.ErrNotFound) {
		return Facts{}, "Ainda não há medições registradas no sistema.", nil
	}
	if err != nil {
		return Facts{}, "", err
	}
	return Facts{
		Consulta: intentDescriptions[IntentCurrentReadings],
		Resultados: []Result{
			{Rotulo: "Temperatura", Valor: metrics[storage.MetricTemperature].format(m.Temperature)},
			{Rotulo: "Umidade", Valor: metrics[storage.MetricHumidity].format(m.Humidity)},
			{Rotulo: "Luminosidade", Valor: metrics[storage.MetricLight].format(m.Light)},
			{Rotulo: "Última medição", Valor: fmtTime(m.MeasuredAt, s.loc) + " (" + fmtAge(s.now().Sub(m.MeasuredAt)) + ")"},
		},
	}, "", nil
}

func (s *Service) aggregate(ctx context.Context, q Query) (Facts, string, error) {
	st, err := s.store.Stats(ctx, q.Period.Range)
	if err != nil {
		return Facts{}, "", err
	}
	if st.Count == 0 {
		return Facts{}, noDataText(q.Period), nil
	}

	info := metrics[q.Metric]
	agg := aggregateOf(st, q.Metric)
	facts := Facts{Consulta: intentDescriptions[q.Intent], Periodo: q.Period.Label, Amostras: st.Count}

	switch q.Intent {
	case IntentAverageTemperature, IntentAverageHumidity:
		facts.Resultados = []Result{{Rotulo: "Média", Valor: info.format(agg.Avg)}}
	case IntentTemperatureTrend, IntentHumidityTrend:
		if st.Count < 2 {
			return Facts{}, fmt.Sprintf("Há poucas medições %s para estimar a tendência.", q.Period.Label), nil
		}
		facts.Resultados = []Result{
			{Rotulo: "Tendência", Valor: trendWord(agg.SlopePerHour, info.stableRate)},
			{Rotulo: "Variação média por hora", Valor: fmtSigned(agg.SlopePerHour, info.decimals) + info.unit + "/h"},
			{Rotulo: "Média", Valor: info.format(agg.Avg)},
			{Rotulo: "Mínimo", Valor: info.format(agg.Min)},
			{Rotulo: "Máximo", Valor: info.format(agg.Max)},
		}
	}
	return facts, "", nil
}

func (s *Service) extreme(ctx context.Context, q Query) (Facts, string, error) {
	highest := q.Intent != IntentMinTemperature
	e, err := s.store.Extreme(ctx, q.Period.Range, q.Metric, highest)
	if errors.Is(err, storage.ErrNotFound) {
		return Facts{}, noDataText(q.Period), nil
	}
	if err != nil {
		return Facts{}, "", err
	}

	info := metrics[q.Metric]
	adjective := "mínima"
	if highest {
		adjective = "máxima"
	}
	return Facts{
		Consulta: intentDescriptions[q.Intent],
		Periodo:  q.Period.Label,
		Resultados: []Result{
			{Rotulo: info.name + " " + adjective, Valor: info.format(e.Value)},
			{Rotulo: "Registrada em", Valor: fmtTime(e.MeasuredAt, s.loc)},
		},
	}, "", nil
}

func (s *Service) compare(ctx context.Context, q Query) (Facts, string, error) {
	cur, err := s.store.Stats(ctx, q.Period.Range)
	if err != nil {
		return Facts{}, "", err
	}
	base, err := s.store.Stats(ctx, q.Period.Baseline().Range)
	if err != nil {
		return Facts{}, "", err
	}
	if cur.Count == 0 || base.Count == 0 {
		return Facts{}, fmt.Sprintf("Não há medições suficientes para comparar %s com o período anterior.", q.Period.Label), nil
	}

	info := metrics[q.Metric]
	curAvg := aggregateOf(cur, q.Metric).Avg
	baseAvg := aggregateOf(base, q.Metric).Avg
	return Facts{
		Consulta: fmt.Sprintf("comparação da média de %s entre o período atual e o período anterior de mesma duração", info.name),
		Periodo:  q.Period.Label,
		Amostras: cur.Count,
		Resultados: []Result{
			{Rotulo: "Média no período atual", Valor: info.format(curAvg)},
			{Rotulo: "Média no período anterior", Valor: info.format(baseAvg)},
			{Rotulo: "Diferença (atual - anterior)", Valor: info.formatSigned(curAvg - baseAvg)},
		},
	}, "", nil
}

func (s *Service) threshold(ctx context.Context, q Query) (Facts, string, error) {
	st, err := s.store.Stats(ctx, q.Period.Range)
	if err != nil {
		return Facts{}, "", err
	}
	if st.Count == 0 {
		return Facts{}, noDataText(q.Period), nil
	}

	c, err := s.store.Crossings(ctx, q.Period.Range, q.Metric, q.Comparison, q.Threshold)
	if err != nil {
		return Facts{}, "", err
	}

	info := metrics[q.Metric]
	word := "acima de"
	if q.Comparison == storage.Below {
		word = "abaixo de"
	}
	limit := word + " " + info.format(q.Threshold)
	if c.Count == 0 {
		return Facts{}, fmt.Sprintf("Não houve medições de %s %s %s.", info.name, limit, q.Period.Label), nil
	}

	return Facts{
		Consulta: fmt.Sprintf("ocorrências de %s %s", info.name, limit),
		Periodo:  q.Period.Label,
		Amostras: st.Count,
		Resultados: []Result{
			{Rotulo: "Medições fora do limite", Valor: strconv.Itoa(c.Count)},
			{Rotulo: "Primeira ocorrência", Valor: fmtTime(*c.First, s.loc)},
			{Rotulo: "Última ocorrência", Valor: fmtTime(*c.Last, s.loc)},
		},
	}, "", nil
}

func noDataText(p Period) string {
	return fmt.Sprintf("Não há medições registradas %s.", p.Label)
}

func aggregateOf(st storage.Stats, m storage.Metric) storage.Aggregate {
	switch m {
	case storage.MetricTemperature:
		return st.Temperature
	case storage.MetricHumidity:
		return st.Humidity
	}
	return st.Light
}

func trendWord(ratePerHour, band float64) string {
	switch {
	case ratePerHour > band:
		return "subindo"
	case ratePerHour < -band:
		return "descendo"
	}
	return "estável"
}

func fmtNumber(v float64, decimals int) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', decimals, 64), ".", ",", 1)
}

func fmtSigned(v float64, decimals int) string {
	s := fmtNumber(v, decimals)
	if v > 0 {
		return "+" + s
	}
	return s
}

func fmtTime(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("02/01/2006 às 15:04")
}

func fmtAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "há menos de 1 min"
	case d < time.Hour:
		return fmt.Sprintf("há %d min", int(d.Minutes()))
	}
	return fmt.Sprintf("há %d h", int(d.Hours()))
}
