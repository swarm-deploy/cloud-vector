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

	http.HandleFunc("/logs", forwarder.ForwardRequest(cloudruStore))

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    serverAddr,
		Handler: http.DefaultServeMux,
		// Таймауты для graceful shutdown
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		slog.Info("[proxy] server started", "port", serverPort, "target", cfg.Cloudru.Logging.Endpoint)
		if serveErr := server.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			slog.Error("[proxy] server failed", "error", serveErr)
			os.Exit(1)
		}
	}()

	// Ожидаем сигналы для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("[proxy] shutting down server gracefully...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	// Останавливаем сервер, давая время для завершения активных запросов
	if err = server.Shutdown(ctx); err != nil {
		slog.Error("[proxy] server forced to shutdown", "error", err)
		cancel()
		os.Exit(1)
	}
	cancel()

	slog.Info("[proxy] server exited", "addr", serverAddr)
}

func initStore(cfg config.Config) (contracts.Store, error) {
	ctx, cancel := context.WithTimeout(context.Background(), initStoreTimeout)
	defer cancel()

	return cloudru.NewStore(ctx, cfg.Cloudru)
}
