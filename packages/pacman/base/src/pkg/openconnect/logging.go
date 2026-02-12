package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"

//go:generate go run ./vendor/golang.org/x/tools/cmd/stringer -type=LogLevel -trimprefix=LogLevel
type LogLevel int

const (
	LogLevelErr   LogLevel = C.PRG_ERR
	LogLevelInfo  LogLevel = C.PRG_INFO
	LogLevelDebug LogLevel = C.PRG_DEBUG
	LogLevelTrace LogLevel = C.PRG_TRACE
)
