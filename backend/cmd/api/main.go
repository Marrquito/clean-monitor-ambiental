package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"clean-backend/internal/api"
	"clean-backend/internal/chat"
	"clean-backend/internal/config"
	"clean-backend/internal/ingest"
	"clean-backend/internal/simulator"
	"clean-backend/internal/storage"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("aplicação encerrada com erro", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	ai := chat.NewOllamaClient(chat.OllamaConfig{
		BaseURL:    cfg.OllamaURL,
		Model:      cfg.OllamaModel,
		Timeout:    cfg.OllamaTimeout,
		KeepAlive:  cfg.OllamaKeepAlive,
		Think:      cfg.OllamaThink,
		NumPredict: cfg.OllamaNumPredict,
	})

	var source api.SourceStatus = api.NoSource{}
	var ingestDone chan struct{}
	if cfg.SimulatorEnabled {
		sim := simulator.NewSource(cfg.Location, cfg.SimulatorInterval)
		if _, err := simulator.Backfill(ctx, simulator.NewGenerator(cfg.Location), store, cfg.SimulatorInterval, cfg.SimulatorBackfill, time.Now()); err != nil {
			return err
		}
		ingestDone = make(chan struct{})
		go func() {
			defer close(ingestDone)
			if err := ingest.Run(ctx, sim, store); err != nil {
				slog.Error("ingestão encerrada", "error", err)
			}
		}()
		source = sim
		slog.Info("simulador ativo", "intervalo", cfg.SimulatorInterval)
	} else {
		slog.Info("simulador desativado por configuração")
	}

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: api.NewRouter(api.Deps{
			Store:     store,
			Assistant: chat.NewService(store, ai, cfg.Location),
			AI:        ai,
			Source:    source,
			Location:  cfg.Location,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("API ouvindo", "endereco", cfg.HTTPAddr)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("encerrando")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if ingestDone != nil {
		<-ingestDone
	}
	return nil
}
