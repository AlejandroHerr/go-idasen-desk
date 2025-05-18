package idasen

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

type (
	Manager struct {
		runCtx       context.Context
		desks        map[string]*DeskService
		initMutexMap map[string]*sync.Mutex
		logger       *slog.Logger
		newBTClient  NewBTClient
	}
	NewBTClient func(context.Context, string) (BTDesk, error)
)

func NewManager(runCtx context.Context, newBLEClient NewBTClient, logger *slog.Logger) *Manager {
	return &Manager{
		newBTClient:  newBLEClient,
		runCtx:       runCtx,
		desks:        make(map[string]*DeskService),
		initMutexMap: make(map[string]*sync.Mutex),
		logger:       logger.With("component", "idasen-manager"),
	}
}

func (m *Manager) ReadHeight(addr string) (int, error) {
	deskService, err := m.getDesk(addr)
	if err != nil {
		return 0, fmt.Errorf("desk not found: %w", err)
	}

	reading, err := deskService.ReadHeight()
	if err != nil {
		return 0, fmt.Errorf("reading desk state: %w", err)
	}

	return reading, nil
}

func (m *Manager) MoveTo(ctx context.Context, addr string, targetHeight int) (int, error) {
	deskService, err := m.getDesk(addr)
	if err != nil {
		return 0, fmt.Errorf("desk not found: %w", err)
	}

	errCh := make(chan error)
	defer close(errCh)

	go deskService.MoveTo(ctx, errCh, targetHeight)

	err = <-errCh

	if err != nil {
		return 0, fmt.Errorf("moving desk to target height: %w", err)
	}

	height, err := deskService.ReadHeight()
	if err != nil {
		m.logger.ErrorContext(ctx, "Error reading height", slog.String("error", err.Error()))
	}

	return height, nil
}

func (m *Manager) Subscribe(addr string, ch chan<- int) (uuid.UUID, error) {
	deskService, err := m.getDesk(addr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("desk not found: %w", err)
	}

	subscriptionID := deskService.Subscribe(ch)

	m.logger.Info(
		"Subscribed to desk service",
		slog.String("address", addr),
		slog.String("subscriptionID", subscriptionID.String()),
	)

	return subscriptionID, nil
}

func (m *Manager) Unsubscribe(addr string, id uuid.UUID) error {
	deskService, err := m.getDesk(addr)
	if err != nil {
		return fmt.Errorf("desk not found: %w", err)
	}

	deskService.Unsubscribe(id)

	return nil
}

func (m *Manager) Close() error {
	for addr, desk := range m.desks {
		if err := desk.Close(); err != nil {
			return fmt.Errorf("closing desk service for %s: %w", addr, err)
		}
	}

	m.logger.Info("All desk services closed")

	return nil
}

func (m *Manager) getDesk(addr string) (*DeskService, error) {
	if err := m.ensureDeskStarted(m.runCtx, addr); err != nil {
		return nil, fmt.Errorf("desk initialization: %w", err)
	}

	deskService, ok := m.desks[addr]
	if !ok {
		return nil, fmt.Errorf("desk service not found for address %s", addr)
	}

	return deskService, nil
}

func (m *Manager) ensureDeskStarted(ctx context.Context, addr string) error {
	deskMutex, ok := m.initMutexMap[addr]
	if !ok {
		deskMutex = &sync.Mutex{}
		m.initMutexMap[addr] = deskMutex
	}

	deskMutex.Lock()
	defer deskMutex.Unlock()

	// Check if the desk is already initialized
	if _, ok = m.desks[addr]; ok {
		return nil
	}

	bleClient, err := m.newBTClient(ctx, addr)
	if err != nil {
		return fmt.Errorf("creating bluetooth client: %w", err)
	}

	deskService := NewDeskService(addr, bleClient, m.logger)
	if err = deskService.Start(ctx); err != nil {
		return fmt.Errorf("starting desk service: %w", err)
	}

	m.desks[addr] = deskService

	m.logger.InfoContext(ctx, "Desk service initialized", slog.String("address", addr))

	return nil
}
