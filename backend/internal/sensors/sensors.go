package sensors

import (
	"context"
	"time"
)

type Measurement struct {
	MeasuredAt  time.Time `json:"measured_at"`
	Temperature float64   `json:"temperature"`
	Humidity    float64   `json:"humidity"`
	Light       float64   `json:"light"`
	Source      string    `json:"source"`
}

// Source produces measurements for the ingestion pipeline. A future MQTT adapter
// implements the same interface, so the storage flow does not change.
type Source interface {
	Kind() string
	Connected() bool
	Run(ctx context.Context, out chan<- Measurement) error
}
