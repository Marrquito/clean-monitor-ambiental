package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"clean-backend/internal/sensors"
)

type Metric string

const (
	MetricTemperature Metric = "temperature"
	MetricHumidity    Metric = "humidity"
	MetricLight       Metric = "light"
)

// column is the only way a metric reaches SQL, so the query text never contains caller input.
func (m Metric) column() (string, error) {
	switch m {
	case MetricTemperature, MetricHumidity, MetricLight:
		return string(m), nil
	}
	return "", fmt.Errorf("métrica inválida: %q", m)
}

type Comparison string

const (
	Above Comparison = "above"
	Below Comparison = "below"
)

func (c Comparison) operator() (string, error) {
	switch c {
	case Above:
		return ">", nil
	case Below:
		return "<", nil
	}
	return "", fmt.Errorf("comparação inválida: %q", c)
}

type Range struct {
	From time.Time
	To   time.Time
}

type Bucket struct {
	Time        time.Time
	Temperature float64
	Humidity    float64
	Light       float64
}

type Aggregate struct {
	Min          float64
	Max          float64
	Avg          float64
	SlopePerHour float64
}

type Stats struct {
	Count       int
	Temperature Aggregate
	Humidity    Aggregate
	Light       Aggregate
}

type Extreme struct {
	MeasuredAt time.Time
	Value      float64
}

type Crossings struct {
	Count int
	First *time.Time
	Last  *time.Time
}

func (s *Store) InsertMeasurements(ctx context.Context, ms []sensors.Measurement) error {
	times := make([]time.Time, len(ms))
	temps := make([]float64, len(ms))
	hums := make([]float64, len(ms))
	lights := make([]float64, len(ms))
	sources := make([]string, len(ms))
	for i, m := range ms {
		times[i] = m.MeasuredAt
		temps[i] = m.Temperature
		hums[i] = m.Humidity
		lights[i] = m.Light
		sources[i] = m.Source
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO measurements (measured_at, temperature, humidity, light, source)
		SELECT * FROM unnest($1::timestamptz[], $2::float8[], $3::float8[], $4::float8[], $5::text[])
		ON CONFLICT (measured_at) DO NOTHING`,
		times, temps, hums, lights, sources)
	return err
}

func (s *Store) LatestMeasurement(ctx context.Context) (sensors.Measurement, error) {
	var m sensors.Measurement
	err := s.pool.QueryRow(ctx, `
		SELECT measured_at, temperature, humidity, light, source
		FROM measurements ORDER BY measured_at DESC LIMIT 1`,
	).Scan(&m.MeasuredAt, &m.Temperature, &m.Humidity, &m.Light, &m.Source)
	if err == pgx.ErrNoRows {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) LastMeasuredAt(ctx context.Context) (time.Time, bool, error) {
	var t *time.Time
	if err := s.pool.QueryRow(ctx, `SELECT max(measured_at) FROM measurements`).Scan(&t); err != nil {
		return time.Time{}, false, err
	}
	if t == nil {
		return time.Time{}, false, nil
	}
	return *t, true, nil
}

func (s *Store) History(ctx context.Context, r Range, limit int) ([]sensors.Measurement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT measured_at, temperature, humidity, light, source
		FROM measurements
		WHERE measured_at >= $1 AND measured_at < $2
		ORDER BY measured_at ASC
		LIMIT $3`,
		r.From, r.To, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]sensors.Measurement, 0)
	for rows.Next() {
		var m sensors.Measurement
		if err := rows.Scan(&m.MeasuredAt, &m.Temperature, &m.Humidity, &m.Light, &m.Source); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) Buckets(ctx context.Context, r Range, width time.Duration) ([]Bucket, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT date_bin(make_interval(secs => $1), measured_at, TIMESTAMPTZ '2000-01-01 00:00:00+00') AS bucket,
		       avg(temperature), avg(humidity), avg(light)
		FROM measurements
		WHERE measured_at >= $2 AND measured_at < $3
		GROUP BY bucket
		ORDER BY bucket`,
		width.Seconds(), r.From, r.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Bucket, 0)
	for rows.Next() {
		var b Bucket
		if err := rows.Scan(&b.Time, &b.Temperature, &b.Humidity, &b.Light); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) Stats(ctx context.Context, r Range) (Stats, error) {
	var st Stats
	err := s.pool.QueryRow(ctx, `
		SELECT count(*),
		       COALESCE(min(temperature), 0), COALESCE(max(temperature), 0), COALESCE(avg(temperature), 0),
		       COALESCE(regr_slope(temperature, extract(epoch FROM measured_at)::float8) * 3600, 0),
		       COALESCE(min(humidity), 0), COALESCE(max(humidity), 0), COALESCE(avg(humidity), 0),
		       COALESCE(regr_slope(humidity, extract(epoch FROM measured_at)::float8) * 3600, 0),
		       COALESCE(min(light), 0), COALESCE(max(light), 0), COALESCE(avg(light), 0)
		FROM measurements
		WHERE measured_at >= $1 AND measured_at < $2`,
		r.From, r.To,
	).Scan(
		&st.Count,
		&st.Temperature.Min, &st.Temperature.Max, &st.Temperature.Avg, &st.Temperature.SlopePerHour,
		&st.Humidity.Min, &st.Humidity.Max, &st.Humidity.Avg, &st.Humidity.SlopePerHour,
		&st.Light.Min, &st.Light.Max, &st.Light.Avg,
	)
	return st, err
}

func (s *Store) Extreme(ctx context.Context, r Range, m Metric, highest bool) (Extreme, error) {
	col, err := m.column()
	if err != nil {
		return Extreme{}, err
	}
	order := "ASC"
	if highest {
		order = "DESC"
	}
	var e Extreme
	err = s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT measured_at, %[1]s
		FROM measurements
		WHERE measured_at >= $1 AND measured_at < $2
		ORDER BY %[1]s %[2]s, measured_at ASC
		LIMIT 1`, col, order),
		r.From, r.To,
	).Scan(&e.MeasuredAt, &e.Value)
	if err == pgx.ErrNoRows {
		return e, ErrNotFound
	}
	return e, err
}

func (s *Store) Crossings(ctx context.Context, r Range, m Metric, c Comparison, threshold float64) (Crossings, error) {
	col, err := m.column()
	if err != nil {
		return Crossings{}, err
	}
	op, err := c.operator()
	if err != nil {
		return Crossings{}, err
	}
	var res Crossings
	var first, last *time.Time
	err = s.pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT count(*), min(measured_at), max(measured_at)
		FROM measurements
		WHERE measured_at >= $1 AND measured_at < $2 AND %s %s $3`, col, op),
		r.From, r.To, threshold,
	).Scan(&res.Count, &first, &last)
	res.First = first
	res.Last = last
	return res, err
}
