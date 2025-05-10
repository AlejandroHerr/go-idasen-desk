package ble

import (
	"fmt"

	goble "github.com/go-ble/ble"
	"github.com/go-ble/ble/darwin"
)

// DefaultDevice ...
func DefaultDevice(opts ...goble.Option) (goble.Device, error) {
	device, err := darwin.NewDevice(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating darwin device: %w", err)
	}

	return device, nil
}
