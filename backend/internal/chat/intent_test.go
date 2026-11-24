package chat

import (
	"testing"
	"time"
)

var sp = mustLoad("America/Sao_Paulo")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// 2026-10-27 02:00 UTC is 2026-10-26 23:00 in São Paulo: "today" must still be the 26th.
var fixedNow = time.Date(2026, 10, 27, 2, 0, 0, 0, time.UTC)

func TestResolvePeriodRelative(t *testing.T) {
	tests := []struct {
		name     string
		wantFrom time.Time
		wantTo   time.Time
	}{
		{PeriodLastHour, fixedNow.Add(-time.Hour), fixedNow},
		{PeriodLast2Hours, fixedNow.Add(-2 * time.Hour), fixedNow},
		{PeriodLast24Hours, fixedNow.Add(-24 * time.Hour), fixedNow},
		{PeriodToday, time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC), fixedNow},
		{PeriodYesterday, time.Date(2026, 10, 25, 3, 0, 0, 0, time.UTC), time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)},
		{PeriodLast7Days, fixedNow.AddDate(0, 0, -7), fixedNow},
		// 2026-10-26 (local) is a Monday: the week starts that same day.
		{PeriodThisWeek, time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC), fixedNow},
		{PeriodLastWeek, time.Date(2026, 10, 19, 3, 0, 0, 0, time.UTC), time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ResolvePeriod(tt.name, "", "", fixedNow, sp)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if !p.Range.From.Equal(tt.wantFrom) || !p.Range.To.Equal(tt.wantTo) {
				t.Fatalf("intervalo = [%v, %v), esperado [%v, %v)", p.Range.From, p.Range.To, tt.wantFrom, tt.wantTo)
			}
		})
	}
}

func TestResolvePeriodCustom(t *testing.T) {
	p, err := ResolvePeriod(PeriodCustom, "2026-10-20", "2026-10-22", fixedNow, sp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	wantFrom := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 10, 23, 3, 0, 0, 0, time.UTC)
	if !p.Range.From.Equal(wantFrom) || !p.Range.To.Equal(wantTo) {
		t.Fatalf("intervalo = [%v, %v), esperado [%v, %v)", p.Range.From, p.Range.To, wantFrom, wantTo)
	}
}

func TestResolvePeriodThisWeekMidweek(t *testing.T) {
	// Thursday 2026-10-29 15:00 local -> week starts Monday 2026-10-26.
	now := time.Date(2026, 10, 29, 18, 0, 0, 0, time.UTC)
	p, err := ResolvePeriod(PeriodThisWeek, "", "", now, sp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if want := time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC); !p.Range.From.Equal(want) {
		t.Fatalf("início = %v, esperado %v", p.Range.From, want)
	}
}

func TestResolvePeriodCustomWithoutEndDateUsesToday(t *testing.T) {
	p, err := ResolvePeriod(PeriodCustom, "2026-10-20", "", fixedNow, sp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !p.Range.To.Equal(fixedNow) {
		t.Fatalf("fim = %v, esperado %v", p.Range.To, fixedNow)
	}
}

func TestResolvePeriodRejectsInvalid(t *testing.T) {
	tests := []struct {
		name       string
		period     string
		start, end string
	}{
		{"nome desconhecido", "next_year", "", ""},
		{"custom sem datas", PeriodCustom, "", ""},
		{"data inicial malformada", PeriodCustom, "20/10/2026", "2026-10-22"},
		{"data final malformada", PeriodCustom, "2026-10-20", "x"},
		{"intervalo invertido", PeriodCustom, "2026-10-22", "2026-10-20"},
		{"intervalo no futuro", PeriodCustom, "2026-11-01", "2026-11-02"},
		{"intervalo maior que 31 dias", PeriodCustom, "2026-08-01", "2026-10-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ResolvePeriod(tt.period, tt.start, tt.end, fixedNow, sp); err == nil {
				t.Fatal("esperava erro de validação")
			}
		})
	}
}

func TestBaselineHasSameDuration(t *testing.T) {
	p, _ := ResolvePeriod(PeriodLastHour, "", "", fixedNow, sp)
	base := p.Baseline()
	if got := base.Range.To.Sub(base.Range.From); got != time.Hour {
		t.Fatalf("duração da linha de base = %v, esperado 1h", got)
	}
	if !base.Range.To.Equal(p.Range.From) {
		t.Fatalf("linha de base deve terminar onde o período começa")
	}
}

func TestValidateQueryDefaultsAndMetrics(t *testing.T) {
	q, err := ValidateQuery(RawIntent{Intent: IntentMaxTemperature}, fixedNow, sp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if q.Period.Name != PeriodLast24Hours {
		t.Fatalf("período padrão = %q, esperado last_24_hours", q.Period.Name)
	}
	if q.Metric != "temperature" {
		t.Fatalf("métrica = %q, esperado temperature", q.Metric)
	}

	q, err = ValidateQuery(RawIntent{Intent: IntentCurrentReadings, Period: "none"}, fixedNow, sp)
	if err != nil || q.Period.Name != "" {
		t.Fatalf("current_readings não deve resolver período: %+v, %v", q, err)
	}
}

func TestValidateQueryRejectsUnsupportedIntent(t *testing.T) {
	_, err := ValidateQuery(RawIntent{Intent: "drop_table"}, fixedNow, sp)
	if err != ErrUnsupportedIntent {
		t.Fatalf("erro = %v, esperado ErrUnsupportedIntent", err)
	}
}

func TestValidateQueryThreshold(t *testing.T) {
	thr := 80.0
	q, err := ValidateQuery(RawIntent{Intent: IntentThresholdQuery, Metric: "humidity", Operator: "above", Threshold: &thr, Period: "today"}, fixedNow, sp)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if q.Metric != "humidity" || q.Threshold != 80 || q.Comparison != "above" {
		t.Fatalf("consulta de limite incorreta: %+v", q)
	}

	if _, err := ValidateQuery(RawIntent{Intent: IntentThresholdQuery, Metric: "humidity"}, fixedNow, sp); err == nil {
		t.Fatal("limite sem valor deve ser rejeitado")
	}
	if _, err := ValidateQuery(RawIntent{Intent: IntentThresholdQuery, Threshold: &thr}, fixedNow, sp); err == nil {
		t.Fatal("limite sem métrica deve ser rejeitado")
	}
	if _, err := ValidateQuery(RawIntent{Intent: IntentHistoricalCompare, Metric: "pressure"}, fixedNow, sp); err == nil {
		t.Fatal("métrica desconhecida deve ser rejeitada")
	}
}

func TestValidateQueryRejectsInvalidPeriod(t *testing.T) {
	if _, err := ValidateQuery(RawIntent{Intent: IntentAverageTemperature, Period: "next_year"}, fixedNow, sp); err == nil {
		t.Fatal("período inválido deve ser rejeitado")
	}
}
