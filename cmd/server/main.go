package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"food-ordering-api/internal/api"
	"food-ordering-api/internal/catalog"
	"food-ordering-api/internal/config"
	"food-ordering-api/internal/coupon"
)

func main() {
	// Optional local file (not required in Docker / K8s where env is injected).
	_ = godotenv.Load()

	cfg := config.FromEnv()

	log := newLogger(cfg.LogJSON, cfg.LogLevel)
	slog.SetDefault(log)

	cv, err := coupon.Load(cfg.CouponFiles)
	if err != nil {
		slog.Error("coupon_load_failed", "err", err)
		os.Exit(1)
	}

	srv := &api.Server{
		Log:            log,
		Catalog:        catalog.NewMemory(cfg.ImageBaseURL),
		Coupon:         cv,
		APIKey:         cfg.APIKey,
		MaxBodyBytes:   cfg.MaxBodyBytes,
		RequestTimeout: cfg.RequestTimeout,
	}

	addr := ":" + cfg.Port
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	go func() {
		slog.Info("listening", "addr", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen_failed", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	slog.Info("shutdown_signal_received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown_failed", "err", err)
		os.Exit(1)
	}
	slog.Info("shutdown_complete")
}

func newLogger(json bool, level string) *slog.Logger {
	hopts := &slog.HandlerOptions{Level: parseLevel(level)}
	if json {
		return slog.New(slog.NewJSONHandler(os.Stdout, hopts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, hopts))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
