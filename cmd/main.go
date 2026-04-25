package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/artarts36/go-entrypoint"
	"github.com/swarm-deploy/cloud-vector/internal/config"
	"github.com/swarm-deploy/cloud-vector/internal/forwarder"
	"github.com/swarm-deploy/cloud-vector/internal/store/cloudru"
	"github.com/swarm-deploy/cloud-vector/internal/store/contracts"

	"github.com/caarlos0/env/v11"
)

const (
	initStoreTimeout = time.Minute
	serverPort       = 8080
	serverAddr       = ":8080"
	readTimeout      = 10 * time.Second
	writeTimeout     = 10 * time.Second
	idleTimeout      = 120 * time.Second
	shutdownTimeout  = 30 * time.Second
)

func main() {
	var cfg config.Config

	if err := env.Parse(&cfg); err != nil {
		slog.Error("[proxy] parse config", slog.Any("err", err))
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Proxy.Log.Level,
	})))

	cloudruStore, err := initStore(cfg)
	if err != nil {
		slog.Error("[proxy] failed to init iam client", slog.Any("err", err))
		os.Exit(1)
	}

	forward := forwarder.NewForwarder(cloudruStore)

	http.HandleFunc("/logs", forward.Forward)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    serverAddr,
		Handler: http.DefaultServeMux,
		// Таймауты для graceful shutdown
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	err = entrypoint.Run([]entrypoint.Entrypoint{
		{
			Name: "forwarder",
			Run: func(context.Context) error {
				return nil
			},
			Stop: func(ctx context.Context) error {
				forward.Stop()

				return nil
			},
		},
		entrypoint.HTTPServer("server", server),
	})
	if err != nil {
		slog.Error("[main] failed to entrypoints", slog.Any("err", err))
	}
}

func initStore(cfg config.Config) (contracts.Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), initStoreTimeout)
	defer cancel()

	return cloudru.NewStore(ctx, cfg.Cloudru)
}
