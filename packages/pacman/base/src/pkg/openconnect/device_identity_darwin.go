package openconnect

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <IOKit/IOKitKeys.h>

static int pacman_platform_uuid(char *buffer, CFIndex size) {
	CFMutableDictionaryRef matching = IOServiceMatching("IOPlatformExpertDevice");
	if (!matching)
		return 0;
	io_service_t service = IOServiceGetMatchingService(kIOMainPortDefault, matching);
	if (!service)
		return 0;
	CFTypeRef value = IORegistryEntryCreateCFProperty(service,
		CFSTR(kIOPlatformUUIDKey), kCFAllocatorDefault, 0);
	IOObjectRelease(service);
	if (!value)
		return 0;
	int ok = CFGetTypeID(value) == CFStringGetTypeID() &&
		CFStringGetLength((CFStringRef)value) == 36 &&
		CFStringGetCString((CFStringRef)value, buffer, size, kCFStringEncodingUTF8);
	CFRelease(value);
	return ok;
}
*/
import "C"

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/sys/unix"
)

// LocalDeviceIdentity reads this Mac's platform version, model, and device ID.
func LocalDeviceIdentity(ctx context.Context) (DeviceIdentity, error) {
	if err := ctx.Err(); err != nil {
		return DeviceIdentity{}, err
	}
	model, err := unix.Sysctl("hw.model")
	if err != nil {
		return DeviceIdentity{}, err
	}
	version, err := unix.Sysctl("kern.osproductversion")
	if err != nil {
		return DeviceIdentity{}, err
	}
	for strings.Count(version, ".") < 2 {
		version += ".0"
	}
	var buffer [37]C.char
	if C.pacman_platform_uuid(&buffer[0], C.CFIndex(len(buffer))) == 0 {
		return DeviceIdentity{}, errors.New("read platform UUID: IOKit property unavailable or invalid")
	}
	// Cisco hashes the canonical uppercase hardware UUID, without a newline.
	uuid := []byte(strings.ToUpper(C.GoString(&buffer[0])))
	uniqueID := sha256.Sum256(uuid)
	return DeviceIdentity{
		PlatformVersion: version, DeviceType: model,
		UniqueID: strings.ToUpper(hex.EncodeToString(uniqueID[:])),
	}, nil
}
