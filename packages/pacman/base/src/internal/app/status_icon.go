package app

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import "status_icon.h"
*/
import "C"

import (
	"runtime"

	"github.com/gilliginsisland/pacman/pkg/dialer"
	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type statusAnimation struct {
	obj *C.struct_PACmanStatusAnimation
}

func newStatusAnimation(status *appkit.StatusItem) *statusAnimation {
	animation := &statusAnimation{}
	status.WithPointer(func(obj appkit.StatusItemRef) {
		animation.obj = C.pacman_status_animation_new((*C.struct_NSStatusItem)(obj))
	})
	runtime.AddCleanup(animation, core.Release, core.ObjectRef(animation.obj))
	return animation
}

func (animation *statusAnimation) updateState(old, next dialer.ConnectionState) {
	defer runtime.KeepAlive(animation)
	C.pacman_status_animation_update_state(animation.obj, C.PACmanStatusState(old), C.PACmanStatusState(next))
}

func (animation *statusAnimation) addInput(delta int) {
	defer runtime.KeepAlive(animation)
	C.pacman_status_animation_add_input(animation.obj, C.NSInteger(delta))
}
