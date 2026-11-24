package simulator

import (
	"context"
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"clean-backend/internal/sensors"
)

const kind = "simulated"

// Generator derives readings from the instant of measurement only, so the same
// timestamp always yields the same values. That keeps backfilled history
// consistent with live data and makes restarts idempotent.
type Generator struct {
	loc *time.Location
}

func NewGenerator(loc *time.Location) Generator {
	return Generator{loc: loc}
}

func (g Generator) At(t time.Time) sensors.Measurement {
	local := t.In(g.loc)
	hour := float64(local.Hour()) + float64(local.Minute())/60 + float64(local.Second())/3600
	minutes := float64(t.Unix()) / 60

	dayCycle := math.Sin(2 * math.Pi * (hour - 9) / 24)
	drift := math.Sin(2 * math.Pi * minutes / (60 * 48))
	wobble := 0.25*math.Sin(2*math.Pi*minutes/97) + 0.15*math.Sin(2*math.Pi*minutes/31)

	temperature := 23 + 4.5*dayCycle + 1.2*drift + wobble
	humidity := clamp(60-12*dayCycle-3*drift+2*wobble, 30, 90)

	return sensors.Measurement{
		MeasuredAt:  t.UTC(),
		Temperature: round(temperature, 1),
		Humidity:    round(humidity, 1),
		Light:       round(lightLux(hour, minutes, wobble), 0),
		Source:      kind,
	}
}

func lightLux(hour, minutes, wobble float64) float64 {
	const sunrise, sunset = 6.0, 18.5
	if hour < sunrise || hour > sunset {
		return math.Max(0, 2+wobble)
	}
	daylight := math.Sin(math.Pi * (hour - sunrise) / (sunset - sunrise))
	cloud := 0.75 + 0.25*math.Sin(2*math.Pi*minutes/240)
	return math.Max(0, 900*math.Pow(daylight, 1.3)*cloud+wobble*5)
}

type Sink interface {
	LastMeasuredAt(ctx context.Context) (time.Time, bool, error)
	InsertMeasurements(ctx context.Context, ms []sensors.Measurement) error
}

// Backfill fills the gap between the last stored measurement and now, limited to
// the last `window`, so a fresh database always holds a full day of history.
func Backfill(ctx context.Context, gen Generator, sink Sink, interval, window time.Duration, now time.Time) (int, error) {
	start := now.Add(-window)
	last, found, err := sink.LastMeasuredAt(ctx)
	if err != nil {
		return 0, err
	}
	if found && last.After(start) {
		start = last.Add(interval)
	}

	const batchSize = 1000
	batch := make([]sensors.Measurement, 0, batchSize)
	total := 0
	for t := start.Truncate(interval); t.Before(now); t = t.Add(interval) {
		if t.Before(start) {
			continue
		}
		batch = append(batch, gen.At(t))
		if len(batch) == batchSize {
			if err := sink.InsertMeasurements(ctx, batch); err != nil {
				return total, err
			}
			total += len(batch)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := sink.InsertMeasurements(ctx, batch); err != nil {
			return total, err
		}
		total += len(batch)
	}
	slog.Info("simulator backfill finished", "measurements", total, "from", start, "to", now)
	return total, nil
}

type Source struct {
	gen       Generator
	interval  time.Duration
	connected atomic.Bool
}

func NewSource(loc *time.Location, interval time.Duration) *Source {
	return &Source{gen: NewGenerator(loc), interval: interval}
}

func (s *Source) Kind() string {
	return kind
}

func (s *Source) Connected() bool {
	return s.connected.Load()
}

func (s *Source) Run(ctx context.Context, out chan<- sensors.Measurement) error {
	s.connected.Store(true)
	defer s.connected.Store(false)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	emit := func(t time.Time) bool {
		select {
		case out <- s.gen.At(t.Truncate(s.interval)):
			return true
		case <-ctx.Done():
			return false
		}
	}

	if !emit(time.Now()) {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case t := <-ticker.C:
			if !emit(t) {
				return nil
			}
		}
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, v))
}

func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}
