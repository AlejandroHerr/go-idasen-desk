package ble

import (
	"fmt"

	goble "github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func DefaultDevice(opts ...goble.Option) (goble.Device, error) {
	device, err := linux.NewDevice(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating linux device: %w", err)
	}

	return device, nil
}
