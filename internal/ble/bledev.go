package ble

import (
	goble "github.com/go-ble/ble"
)

func NewDevice(_ string, opts ...goble.Option) (goble.Device, error) {
	return DefaultDevice(opts...)
}
