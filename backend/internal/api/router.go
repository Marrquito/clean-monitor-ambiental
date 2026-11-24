package api

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"clean-backend/internal/chat"
	"clean-backend/internal/sensors"
	"clean-backend/internal/storage"
)

type Deps struct {
	Store     DataStore
	Assistant Assistant
	AI        AIStatus
	Source    SourceStatus
	Location  *time.Location
}

type SourceStatus interface {
	Kind() string
	Connected() bool
}

type NoSource struct{}

func (NoSource) Kind() string    { return "none" }
func (NoSource) Connected() bool { return false }

func NewRouter(d Deps) *gin.Engine {
	h := &handler{deps: d, now: time.Now}

	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	api := r.Group("/api")
	api.GET("/health", h.health)
	api.GET("/sensors/latest", h.latest)
	api.GET("/sensors/history", h.history)
	api.GET("/dashboard/summary", h.summary)
	api.POST("/chat", h.chat)
	return r
}

type DataStore interface {
	Ping(ctx context.Context) error
	LatestMeasurement(ctx context.Context) (sensors.Measurement, error)
	LastMeasuredAt(ctx context.Context) (time.Time, bool, error)
	History(ctx context.Context, r storage.Range, limit int) ([]sensors.Measurement, error)
	Buckets(ctx context.Context, r storage.Range, width time.Duration) ([]storage.Bucket, error)
	Stats(ctx context.Context, r storage.Range) (storage.Stats, error)
}

type Assistant interface {
	Answer(ctx context.Context, message string) (chat.Answer, error)
}

type AIStatus interface {
	ModelInstalled(ctx context.Context) (bool, error)
	Model() string
}
