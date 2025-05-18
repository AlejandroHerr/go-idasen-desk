package ble

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"github.com/AlejandroHerr/go-idasen-desk/internal/idasen"
	goble "github.com/go-ble/ble"
)

const (
	heightCharUUID   = "99fa0021338a10248a49009c0215f78a"
	controlCharrUUID = "99fa0002338a10248a49009c0215f78a"
	offsetHeight     = 6150
	uint32Size       = 4 // 32 bits
)

var (
	moveUpCmd                 = []byte{0x47, 0x00} //nolint:gochecknoglobals //needs to be a const
	moveDownCmd               = []byte{0x46, 0x00} //nolint:gochecknoglobals //needs to be a const
	stopCmd                   = []byte{0xFF, 0x00} //nolint:gochecknoglobals //needs to be a const
	_           idasen.BTDesk = (*DeskClient)(nil)
)

type (
	DeskClient struct {
		client      goble.Client
		controlChar *goble.Characteristic
		heightChar  *goble.Characteristic
		logger      *slog.Logger
	}
)

func NewDeskClientFunc(device goble.Device, logger *slog.Logger) idasen.NewBTClient {
	return func(ctx context.Context, addr string) (idasen.BTDesk, error) {
		return NewDeskClient(ctx, addr, device, logger)
	}
}

func NewDeskClient(ctx context.Context, addr string, device goble.Device, logger *slog.Logger) (*DeskClient, error) {
	bleAddr := goble.NewAddr(addr)

	client, err := device.Dial(ctx, bleAddr)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", bleAddr.String(), err)
	}

	services, err := client.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("discovering services: %w", err)
	}

	var controlChar, heightChar *goble.Characteristic

	for _, service := range services {
		chars, err := client.DiscoverCharacteristics( //nolint:govet,shadow // this is the correct way to use it
			nil,
			service,
		)
		if err != nil {
			return nil, fmt.Errorf("discovering characteristics for service %s: %w", service.UUID.String(), err)
		}

		for _, char := range chars {
			if char.UUID.String() == controlCharrUUID {
				controlChar = char

				break
			} else if char.UUID.String() == heightCharUUID {
				heightChar = char

				break
			}
		}
	}

	if controlChar == nil {
		if err = client.CancelConnection(); err != nil {
			logger.ErrorContext(
				ctx,
				"Cancelling contection",
				slog.String("error", err.Error()),
				slog.String("address", addr),
			)
		}

		return nil, fmt.Errorf("control characteristic '%s' not found", controlCharrUUID)
	}

	if heightChar == nil {
		if err = client.CancelConnection(); err != nil {
			logger.ErrorContext(
				ctx,
				"Cancelling contection",
				slog.String("error", err.Error()),
				slog.String("address", addr),
			)
		}

		return nil, fmt.Errorf("height characteristic '%s' not found", heightCharUUID)
	}

	return &DeskClient{
		client:      client,
		controlChar: controlChar,
		heightChar:  heightChar,
		logger: logger.With(
			slog.String("component", "ble-desk-client"),
			slog.String("address", client.Addr().String()),
		),
	}, nil
}

func (c *DeskClient) ReadHeight() (int, error) {
	value, err := c.client.ReadCharacteristic(c.heightChar)
	if err != nil {
		return 0, fmt.Errorf("reading height characteristic: %w", err)
	}

	height, err := c.parseHeight(value)
	if err != nil {
		return 0, fmt.Errorf("parsing height and speed: %w", err)
	}

	return height, nil
}

func (c *DeskClient) MoveUp() error {
	if err := c.client.WriteCharacteristic(c.controlChar, moveUpCmd, true); err != nil {
		return fmt.Errorf(
			"writing command 0x%X to characteristic %s: %w",
			moveUpCmd,
			c.controlChar.UUID.String(),
			err,
		)
	}

	return nil
}

func (c *DeskClient) MoveDown() error {
	if err := c.client.WriteCharacteristic(c.controlChar, moveDownCmd, true); err != nil {
		return fmt.Errorf(
			"writing command 0x%X to characteristic %s: %w",
			moveDownCmd,
			c.controlChar.UUID.String(),
			err,
		)
	}

	return nil
}

func (c *DeskClient) Stop() error {
	if err := c.client.WriteCharacteristic(c.controlChar, stopCmd, true); err != nil {
		return fmt.Errorf(
			"writing command 0x%X to characteristic %s: %w",
			stopCmd,
			c.controlChar.UUID.String(),
			err,
		)
	}

	return nil
}

func (c *DeskClient) Subscribe(ch chan<- int) error {
	notificationHandler := func(data []byte) {
		height, err := c.parseHeight(data)
		if err != nil {
			c.logger.Warn("Parsing height", slog.String("error", err.Error()))
			return
		}

		ch <- height
	}

	if err := c.client.Subscribe(c.heightChar, false, notificationHandler); err != nil {
		return fmt.Errorf(
			"subscribing to height characteristic %s: %w",
			c.heightChar.UUID.String(),
			err,
		)
	}

	return nil
}

func (c *DeskClient) Unsubscribe() error {
	if err := c.client.Unsubscribe(c.heightChar, false); err != nil {
		return fmt.Errorf(
			"unsubscribing from height characteristic %s: %w",
			c.heightChar.UUID.String(),
			err,
		)
	}

	return nil
}

func (c *DeskClient) Close() error {
	if err := c.client.CancelConnection(); err != nil {
		return fmt.Errorf("canceling connection: %w", err)
	}

	return nil
}

func (c *DeskClient) parseHeight(data []byte) (int, error) {
	if len(data) < uint32Size {
		return 0, errors.New("invalid data length")
	}

	height := int(binary.LittleEndian.Uint16(data[0:2])) + offsetHeight

	return height, nil
}
