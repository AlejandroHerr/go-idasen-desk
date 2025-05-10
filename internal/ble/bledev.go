package ble

import (
	goble "github.com/go-ble/ble"
)

func NewDevice(_ string, opts ...goble.Option) (d goble.Device, err error) {
	return DefaultDevice(opts...)
}
