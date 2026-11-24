package ingest

import (
	"context"
	"log/slog"

	"clean-backend/internal/sensors"
)

type Writer interface {
	InsertMeasurements(ctx context.Context, ms []sensors.Measurement) error
}

// Run persists every measurement from src until ctx is cancelled.
func Run(ctx context.Context, src sensors.Source, w Writer) error {
	ch := make(chan sensors.Measurement)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		errCh <- src.Run(ctx, ch)
	}()

	for m := range ch {
		if err := w.InsertMeasurements(ctx, []sensors.Measurement{m}); err != nil {
			slog.Error("falha ao salvar medição", "source", src.Kind(), "error", err)
		}
	}
	return <-errCh
}
