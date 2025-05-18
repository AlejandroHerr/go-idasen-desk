package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlejandroHerr/go-common/pkg/logging"
	"github.com/AlejandroHerr/go-idasen-desk/internal/ble"
	"github.com/AlejandroHerr/go-idasen-desk/internal/config"
	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	"github.com/AlejandroHerr/go-idasen-desk/internal/restapi"
	"github.com/AlejandroHerr/go-idasen-desk/version"
	goble "github.com/go-ble/ble"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancelCtx()

	logger := logging.NewLogger(
		logging.WithApp("go-idasen-desk-cli"),
		logging.WithEnvironment(version.GetEnvironment()),
		logging.WithVersion(version.GetVersion()),
		logging.WithCommit(version.GetCommit()),
		logging.WithBuildTime(version.GetBuildTime()),
		logging.WithGoVersion(version.GetGoVersion()),
	)

	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "Error occurred", slog.String("error", err.Error()))

		cancelCtx()

		os.Exit(1) //nolint:gocritic // it is ok
	}
}

func run(pctx context.Context, logger *slog.Logger) error {
	ctx, cancelCtx := context.WithCancel(pctx)
	defer cancelCtx()

	cfg, err := loadConfig(logger)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dev, err := ble.NewDevice("default")
	if err != nil {
		return fmt.Errorf("new device: %w", err)
	}

	defer func() {
		logger.InfoContext(ctx, "Shutting down device...")

		if err = dev.Stop(); err != nil {
			logger.ErrorContext(ctx, "Error stopping device", slog.String("error", err.Error()))
		}
	}()

	goble.SetDefaultDevice(dev)

	manager := idasen.NewManager(ctx, ble.NewDeskClientFunc(dev, logger), logger)
	defer func() {
		logger.InfoContext(ctx, "Shutting down manager...")

		if err = manager.Close(); err != nil {
			logger.ErrorContext(ctx, "Error closing manager", slog.String("error", err.Error()))
		}
	}()

	handler := restapi.NewHandler(cfg.AuthTokens, manager, logger)

	serverResult := make(chan error, 1)
	defer close(serverResult)

	go startServer(ctx, cfg.Port, serverResult, handler, logger)

	select {
	case err = <-serverResult:
		if err != nil {
			return fmt.Errorf("starting server: %w", err)
		}
	case <-ctx.Done():
		logger.InfoContext(ctx, "Shutting down...")

		return nil
	}

	return nil
}

const defaultConfigPath = "/etc/go-idasen-desk/config.yaml"

func loadConfig(logger *slog.Logger) (*config.RestConfig, error) {
	configPath := flag.String("config", defaultConfigPath, "Path to the config file")
	flag.Parse()

	cfg, err := config.Load(*configPath, logger)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	return &cfg.Rest, nil
}

const defaultReadHeaderTimeout = 5 * time.Minute

func startServer(ctx context.Context, port int, resultCh chan<- error, handler http.Handler, logger *slog.Logger) {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	defer func() {
		logger.InfoContext(ctx, "Shutting down server...")

		if err := server.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "Error shutting down server", slog.String("error", err.Error()))
		}
	}()

	errCh := make(chan error, 1)

	go func() {
		logger.InfoContext(ctx, "Starting server...", slog.Int("port", port))

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "Error starting server", slog.String("error", err.Error()))
			errCh <- err

			return
		}
	}()

	select {
	case err := <-errCh:
		resultCh <- err
	case <-ctx.Done():
	}
}
