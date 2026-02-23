package app

import (
	"unsafe"
)

type SparkleUpdateAction struct{}

func (s SparkleUpdateAction) Action() string {
	return "sparkle_check_updates"
}

func (s SparkleUpdateAction) Data() unsafe.Pointer {
	return nil
}
