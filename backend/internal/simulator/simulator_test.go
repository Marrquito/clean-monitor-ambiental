package simulator

import (
	"context"
	"testing"
	"time"

	"clean-backend/internal/sensors"
)

var testLoc = mustLoad("America/Sao_Paulo")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type memorySink struct {
	last    time.Time
	hasLast bool
	rows    map[time.Time]sensors.Measurement
}

func newMemorySink() *memorySink {
	return &memorySink{rows: map[time.Time]sensors.Measurement{}}
}

func (s *memorySink) LastMeasuredAt(context.Context) (time.Time, bool, error) {
	return s.last, s.hasLast, nil
}

func (s *memorySink) InsertMeasurements(_ context.Context, ms []sensors.Measurement) error {
	for _, m := range ms {
		if _, dup := s.rows[m.MeasuredAt]; dup {
			continue
		}
		s.rows[m.MeasuredAt] = m
		if !s.hasLast || m.MeasuredAt.After(s.last) {
			s.last, s.hasLast = m.MeasuredAt, true
		}
	}
	return nil
}

func TestGeneratorIsDeterministicAndPlausible(t *testing.T) {
	gen := NewGenerator(testLoc)
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 2*24*60; i += 7 {
		ts := start.Add(time.Duration(i) * time.Minute)
		a, b := gen.At(ts), gen.At(ts)
		if a != b {
			t.Fatalf("geração não determinística em %v", ts)
		}
		if a.Temperature < 10 || a.Temperature > 40 {
			t.Fatalf("temperatura implausível %.1f em %v", a.Temperature, ts)
		}
		if a.Humidity < 30 || a.Humidity > 90 {
			t.Fatalf("umidade fora da faixa %.1f em %v", a.Humidity, ts)
		}
		if a.Light < 0 {
			t.Fatalf("luminosidade negativa em %v", ts)
		}
	}
}

func TestGeneratorHasDailyCycle(t *testing.T) {
	gen := NewGenerator(testLoc)
	night := gen.At(time.Date(2026, 10, 27, 6, 0, 0, 0, testLoc))
	afternoon := gen.At(time.Date(2026, 10, 27, 15, 0, 0, 0, testLoc))
	if afternoon.Temperature <= night.Temperature {
		t.Fatalf("tarde (%.1f) deveria ser mais quente que madrugada (%.1f)", afternoon.Temperature, night.Temperature)
	}
	if afternoon.Light <= night.Light {
		t.Fatalf("tarde deveria ter mais luz que madrugada")
	}
}

func TestBackfillFillsEmptyWindow(t *testing.T) {
	sink := newMemorySink()
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	n, err := Backfill(context.Background(), NewGenerator(testLoc), sink, time.Minute, 24*time.Hour, now)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if n != 24*60 {
		t.Fatalf("inseridas %d medições, esperado %d", n, 24*60)
	}
}

func TestBackfillDoesNotDuplicateOnRestart(t *testing.T) {
	sink := newMemorySink()
	gen := NewGenerator(testLoc)
	ctx := context.Background()
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

	if _, err := Backfill(ctx, gen, sink, time.Minute, 24*time.Hour, now); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	later := now.Add(30 * time.Minute)
	n, err := Backfill(ctx, gen, sink, time.Minute, 24*time.Hour, later)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if n != 30 {
		t.Fatalf("reinício inseriu %d medições, esperado apenas as 30 faltantes", n)
	}
	if len(sink.rows) != 24*60+30 {
		t.Fatalf("total = %d, esperado %d sem duplicatas", len(sink.rows), 24*60+30)
	}
}
