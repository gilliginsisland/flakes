//go:build !darwin

package openconnect

import "context"

// LocalDeviceIdentity returns an empty identity on unsupported platforms.
func LocalDeviceIdentity(context.Context) (DeviceIdentity, error) {
	return DeviceIdentity{}, nil
}
