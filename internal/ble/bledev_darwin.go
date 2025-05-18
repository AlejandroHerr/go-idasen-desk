package ble

import (
	"fmt"

	goble "github.com/go-ble/ble"
	"github.com/go-ble/ble/darwin"
)

func DefaultDevice(opts ...goble.Option) (*darwin.Device, error) {
	device, err := darwin.NewDevice(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating darwin device: %w", err)
	}

	return device, nil
}
