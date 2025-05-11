package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/AlejandroHerr/go-idasen-desk/internal/ble"
	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	"github.com/AlejandroHerr/go-idasen-desk/internal/logger"
	"github.com/AlejandroHerr/go-idasen-desk/version"
	goble "github.com/go-ble/ble"
)

func main() {
	ctx := context.Background()
	logger := logger.New(
		logger.WithApp("go-idasen-desk-scanner"),
		logger.WithEnvironment(version.GetEnvironment()),
	)

	if err := run(ctx, logger); err != nil {
		logger.ErrorContext(ctx, "Error occurred", slog.String("error", err.Error()))

		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	dev, err := ble.NewDevice("default")
	if err != nil {
		return fmt.Errorf("new device: %w", err)
	}

	goble.SetDefaultDevice(dev)

	bleScanner := ble.NewScanner(dev, logger)

	scanner := idasen.NewScanner(bleScanner, logger)

	advs, err := scanner.ScanDesks(ctx)
	if err != nil {
		return fmt.Errorf("scanning desks: %w", err)
	}

	logger.InfoContext(ctx, "Desks found", slog.Int("count", len(advs)))

	for _, adv := range advs {
		logger.InfoContext(ctx, "Desk found", slog.String("address", adv.Addr), slog.String("name", adv.Name))
	}

	return nil
}
