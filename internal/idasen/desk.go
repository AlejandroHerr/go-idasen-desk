package idasen

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultMargin       = 10
	defaultTimeout      = 30 * time.Second
	defaultPollInterval = 100 * time.Millisecond
	minDeskHeight       = 6150
	maxDeskHeight       = 12700
	dirUp               = 1
	dirDown             = -1
)

var (
	ErrNotRunning    = errors.New("desk service is not running")
	ErrInvalidHeight = fmt.Errorf("invalid height, range is %d - %d", minDeskHeight, maxDeskHeight)
	ErrCancelled     = errors.New("operation cancelled")
	ErrTimeout       = errors.New("operation timed out")
)

type (
	BTDesk interface {
		ReadHeight() (int, error)
		MoveUp() error
		MoveDown() error
		Stop() error
		Subscribe(ch chan<- int) error
		Unsubscribe() error
		Close() error
	}
	DeskService struct {
		uuid           string
		client         BTDesk
		logger         *slog.Logger
		options        *DeskServiceOptions
		isRunning      bool
		isRunningMutex sync.RWMutex
		height         int
		readMutex      sync.RWMutex
		moveToCmdCh    chan MoveToCmd
		subscribers    []Subscription
	}
	DeskServiceOptions struct {
		margin       int
		timeout      time.Duration
		pollInterval time.Duration
	}
	MoveToCmd struct {
		TargetHeight int
		Ctx          context.Context
		ResultCh     chan<- error
	}
	DeskServiceOption func(*DeskServiceOptions)
	Subscription      struct {
		id string
		ch chan<- int
	}
)

func NewDeskService(uuid string, client BTDesk, logger *slog.Logger, opts ...DeskServiceOption) *DeskService {
	options := &DeskServiceOptions{
		margin:       defaultMargin,
		timeout:      defaultTimeout,
		pollInterval: defaultPollInterval,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &DeskService{
		uuid:           uuid,
		height:         0,
		readMutex:      sync.RWMutex{},
		options:        options,
		moveToCmdCh:    make(chan MoveToCmd),
		isRunning:      false,
		isRunningMutex: sync.RWMutex{},
		subscribers:    []Subscription{},
		client:         client,
		logger: logger.With(
			slog.String("component", "idasen-desk-service"),
			slog.String("uuid", uuid),
		),
	}
}

func (s *DeskService) Start(ctx context.Context) error {
	isRunning := s.readIsRunning()
	if isRunning {
		s.logger.InfoContext(ctx, "Desk service is already running")
		return nil
	}

	errCh := make(chan error)
	defer close(errCh)

	go s.run(ctx, errCh)

	err := <-errCh
	if err != nil {
		return fmt.Errorf("starting desk service: %w", err)
	}

	s.logger.InfoContext(ctx, "Desk service running")

	return nil
}

func (s *DeskService) ReadHeight() (int, error) {
	if !s.readIsRunning() {
		return 0, ErrNotRunning
	}

	return s.readHeight(), nil
}

func (s *DeskService) MoveTo(ctx context.Context, resultCh chan error, targetHeight int) {
	if !s.readIsRunning() {
		resultCh <- ErrNotRunning

		return
	}

	if targetHeight < minDeskHeight || targetHeight > maxDeskHeight {
		resultCh <- ErrInvalidHeight

		return
	}

	s.moveToCmdCh <- MoveToCmd{
		TargetHeight: targetHeight,
		ResultCh:     resultCh,
		Ctx:          ctx,
	}
}

func (s *DeskService) Subscribe(ch chan<- int) uuid.UUID {
	id := uuid.New()

	s.subscribers = append(s.subscribers, Subscription{
		id: id.String(),
		ch: ch,
	})

	return id
}

func (s *DeskService) Unsubscribe(id uuid.UUID) {
	for i, sub := range s.subscribers {
		if sub.id == id.String() {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}
}

func (s *DeskService) Close() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("closing desk client: %w", err)
	}

	s.logger.Info("Desk service closed")

	return nil
}

func (s *DeskService) inTargetRange(currentHeight, targetHeight int) bool {
	return currentHeight >= targetHeight-s.options.margin && currentHeight <= targetHeight+s.options.margin
}

func (s *DeskService) run(ctx context.Context, errCh chan<- error) {
	defer func() {
		s.updateIsRunning(false)
	}()

	s.updateIsRunning(true)

	height, err := s.client.ReadHeight()
	if err != nil {
		errCh <- fmt.Errorf("reading initial height: %w", err)
		return
	}

	s.updateHeight(height)

	updateCh := make(chan int)
	defer close(updateCh)

	err = s.client.Subscribe(updateCh)
	if err != nil {
		errCh <- fmt.Errorf("subscribing to height updates: %w", err)
		return
	}

	defer func() {
		if err = s.client.Unsubscribe(); err != nil {
			s.logger.ErrorContext(
				ctx,
				"Error unsubscribing from height updates",
				slog.String("error", err.Error()),
			)
		}
	}()

	errCh <- nil

	var (
		moveToCtx    context.Context
		moveToCancel context.CancelFunc
	)

	defer func() {
		if moveToCancel != nil {
			moveToCancel()
		}
	}()

	for {
		select {
		case updatedHeight := <-updateCh:
			s.logger.DebugContext(
				ctx,
				"Received height update",
				slog.Int("height", updatedHeight),
			)

			s.updateHeight(updatedHeight)

			for _, sub := range s.subscribers {
				sub.ch <- updatedHeight
			}

		case moveToCmd := <-s.moveToCmdCh:
			s.logger.DebugContext(
				ctx,
				"Received moveTo command",
				slog.Int("targetHeight", moveToCmd.TargetHeight),
			)

			if moveToCancel != nil {
				s.logger.DebugContext(ctx, "Canceling previous moveTo command")
				moveToCancel()
			}

			moveToCtx, moveToCancel = context.WithTimeout( //nolint:fatcontext // sda
				moveToCmd.Ctx,
				s.options.timeout,
			)

			go s.handleMoveTo(moveToCtx, moveToCmd.TargetHeight, moveToCmd.ResultCh)
		case <-ctx.Done():
			s.logger.InfoContext(ctx, "Desk service stopped")
		}
	}
}

func (s *DeskService) handleMoveTo(ctx context.Context, targetHeight int, resultCh chan<- error) {
	currentHeight := s.readHeight()

	if s.inTargetRange(currentHeight, targetHeight) {
		s.logger.InfoContext(
			ctx,
			"Desk is already in target range",
			slog.Int("currentHeight", currentHeight),
			slog.Int("targetHeight", targetHeight),
		)

		resultCh <- nil

		return
	}

	s.logger.InfoContext(
		ctx,
		"Moving desk",
		slog.Int("from", currentHeight),
		slog.Int("to", targetHeight),
	)

	if err := s.moveToTarget(currentHeight, targetHeight); err != nil {
		resultCh <- fmt.Errorf("moving desk to: %w", err)

		return
	}

	defer func() {
		if err := s.client.Stop(); err != nil {
			s.logger.ErrorContext(ctx, "Error stopping desk", slog.String("error", err.Error()))
		}
	}()

	ticker := time.NewTicker(s.options.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			currentHeight = s.readHeight()

			if s.inTargetRange(currentHeight, targetHeight) {
				s.logger.DebugContext(
					ctx,
					"Desk reached target range",
					slog.Int("currentHeight", currentHeight),
					slog.Int("targetHeight", targetHeight),
				)

				resultCh <- nil

				return
			}

			if err := s.moveToTarget(currentHeight, targetHeight); err != nil {
				resultCh <- fmt.Errorf("moving desk to target: %w", err)

				return
			}

		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				switch {
				case errors.Is(err, context.Canceled):
					s.logger.DebugContext(ctx, "MoveTo command cancelled")
					resultCh <- ErrCancelled
				case errors.Is(err, context.DeadlineExceeded):
					s.logger.DebugContext(ctx, "MoveTo command timed out")
					resultCh <- ErrTimeout
				default:
					s.logger.ErrorContext(ctx, "MoveTo command error", slog.String("error", err.Error()))
					resultCh <- fmt.Errorf("moveTo command error: %w", err)
				}
			} else {
				resultCh <- nil
			}

			return
		}
	}
}

func (s *DeskService) moveToTarget(currentHeight, targetHeight int) error {
	if targetHeight > currentHeight {
		if err := s.client.MoveUp(); err != nil {
			return fmt.Errorf("moving desk up: %w", err)
		}
	} else {
		if err := s.client.MoveDown(); err != nil {
			return fmt.Errorf("moving desk down: %w", err)
		}
	}

	return nil
}

func (s *DeskService) readHeight() int {
	s.readMutex.RLock()
	defer s.readMutex.RUnlock()

	return s.height
}

func (s *DeskService) updateHeight(reading int) {
	s.readMutex.Lock()
	defer s.readMutex.Unlock()

	s.height = reading
}

func (s *DeskService) readIsRunning() bool {
	s.isRunningMutex.RLock()
	defer s.isRunningMutex.RUnlock()

	return s.isRunning
}

func (s *DeskService) updateIsRunning(isRunning bool) {
	s.isRunningMutex.Lock()
	defer s.isRunningMutex.Unlock()

	s.isRunning = isRunning
}
