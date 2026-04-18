package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swarm-deploy/cloud-vector/internal/config"
	"github.com/swarm-deploy/cloud-vector/internal/forwarder"
	"github.com/swarm-deploy/cloud-vector/internal/store/cloudru"
	"github.com/swarm-deploy/cloud-vector/internal/store/contracts"

	"github.com/caarlos0/env/v11"
)

const initStoreTimeout = time.Minute

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

	http.HandleFunc("/logs", forwarder.ForwardRequest(cloudruStore))

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    ":8080",
		Handler: http.DefaultServeMux,
		// Таймауты для graceful shutdown
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		slog.Info("[proxy] server started", "port", 8080, "target", cfg.Cloudru.Logging.Endpoint)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("[proxy] server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Ожидаем сигналы для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("[proxy] shutting down server gracefully...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем сервер, давая время для завершения активных запросов
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("[proxy] server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("[proxy] server exited")
}

func initStore(cfg config.Config) (contracts.Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), initStoreTimeout)
	defer cancel()

	return cloudru.NewStore(ctx, cfg.Cloudru)
}
