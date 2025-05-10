package idasen

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const scannerDefaultTimeout = 10 * time.Second

type (
	Advertisement struct {
		Name string
		Addr string
	}
	ScannerOptions struct {
		Timeout time.Duration
	}
	BTScanner interface {
		ScanByName(ctx context.Context, name string, timeout time.Duration) ([]Advertisement, error)
	}
	ScannerOptionsFunc func(*ScannerOptions)
	Scanner            struct {
		scanner BTScanner
		options *ScannerOptions
		logger  *slog.Logger
	}
)

func NewScanner(scanner BTScanner, logger *slog.Logger, opts ...ScannerOptionsFunc) *Scanner {
	options := &ScannerOptions{
		Timeout: scannerDefaultTimeout,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Scanner{
		scanner: scanner,
		options: options,
		logger:  logger.With(slog.String("component", "idasen-scanner")),
	}
}

func (s *Scanner) ScanDesks(ctx context.Context) ([]Advertisement, error) {
	s.logger.DebugContext(ctx, "Scanning for desks")

	advs, err := s.scanner.ScanByName(ctx, "^Desk", s.options.Timeout)
	if err != nil {
		return nil, fmt.Errorf("scanning by name: %w", err)
	}

	s.logger.DebugContext(ctx, "Found desks", slog.Int("count", len(advs)))

	return advs, nil
}

func ScannerOptionsWithTimeout(timeout time.Duration) ScannerOptionsFunc {
	return func(o *ScannerOptions) {
		o.Timeout = timeout
	}
}
