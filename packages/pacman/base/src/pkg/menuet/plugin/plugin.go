// Package plugin loads native menu providers bundled with the application.
package plugin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "plugin.h"
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
)

type Plugin interface {
	MenuItems() []*appkit.MenuItem
}

type bundlePlugin struct {
	obj *C.struct_NSObject
}

// Load loads PACmanPlugin providers from the app's own PlugIns directory.
// A missing directory means no plugins. Invalid or unloadable bundles are errors.
func Load() ([]Plugin, error) {
	var message *C.char
	values := C.menuet_plugins_load(&message)
	if message != nil {
		defer C.free(unsafe.Pointer(message))
		return nil, errors.New(C.GoString(message))
	}
	defer C.free(unsafe.Pointer(values.items))
	plugins := make([]Plugin, values.count)
	for index, obj := range unsafe.Slice(values.items, values.count) {
		plugin := &bundlePlugin{obj: obj}
		runtime.AddCleanup(plugin, core.Release, unsafe.Pointer(obj))
		plugins[index] = plugin
	}
	return plugins, nil
}

// MenuItems returns owned wrappers. Keep the plugin alive while its items are
// installed: the menu retains the items, but not their action targets.
func (plugin *bundlePlugin) MenuItems() []*appkit.MenuItem {
	defer runtime.KeepAlive(plugin)
	values := C.menuet_plugin_menu_items(plugin.obj)
	defer C.free(unsafe.Pointer(values.items))
	items := make([]*appkit.MenuItem, values.count)
	for index, obj := range unsafe.Slice(values.items, values.count) {
		items[index] = appkit.MenuItemFromPointer(unsafe.Pointer(obj))
	}
	return items
}
