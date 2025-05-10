package ble

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	goble "github.com/go-ble/ble"
)

type Scanner struct {
	device goble.Device
	logger *slog.Logger
}

var _ idasen.BTScanner = (*Scanner)(nil)

func NewScanner(device goble.Device, logger *slog.Logger) *Scanner {
	return &Scanner{
		device: device,
		logger: logger.With(slog.String("component", "ble-scanner")),
	}
}

func (s *Scanner) ScanByName(
	ctx context.Context,
	nameRegexp string,
	timeout time.Duration,
) ([]idasen.Advertisement, error) {
	rgxp, err := regexp.Compile(nameRegexp)
	if err != nil {
		return nil, fmt.Errorf("compiling regexp: %w", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	s.logger.DebugContext(ctxWithTimeout, "Starting scan")

	advs := make([]idasen.Advertisement, 0)
	advsMap := make(map[string]bool)

	err = s.device.Scan(ctxWithTimeout, false, func(a goble.Advertisement) {
		if _, ok := advsMap[a.Addr().String()]; ok {
			return
		}

		// Ignore duplicates
		advsMap[a.Addr().String()] = true

		matches := rgxp.MatchString(a.LocalName())

		s.logger.DebugContext(
			ctx,
			"Advertisement found",
			slog.Bool("matches", matches),
			slog.String("address", a.Addr().String()),
			slog.String("name", a.LocalName()),
		)

		if matches {
			advs = append(advs, idasen.Advertisement{
				Name: a.LocalName(),
				Addr: a.Addr().String(),
			})
		}
	})

	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("scanning: %w", err)
	}

	return advs, nil
}
