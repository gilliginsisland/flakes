package plugin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>
#import "plugin.h"
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/menuet/appkit"
	"github.com/gilliginsisland/pacman/pkg/menuet/core"
	"github.com/gilliginsisland/pacman/pkg/menuet/foundation"
)

type Plugin struct {
	obj *C.struct_NSObject
}

// Load loads PACmanPlugin providers from the app's own PlugIns directory.
// A missing directory means no plugins. Invalid or unloadable bundles are errors.
func Load() ([]*Plugin, error) {
	dir := foundation.MainBundle().BuiltInPlugInsPath()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var plugins []*Plugin
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || filepath.Ext(name) != ".bundle" {
			continue
		}
		bundle := foundation.BundleWithPath(filepath.Join(dir, name))
		if bundle == nil {
			return nil, fmt.Errorf("load plugin %s: invalid bundle", name)
		}
		if err := bundle.Load(); err != nil {
			return nil, fmt.Errorf("load plugin %s: %w", name, err)
		}
		var message *C.char
		obj := C.pacman_plugin_new(bundle.PrincipalClass(), &message)
		if message != nil {
			err := fmt.Errorf("plugin %s: %s", name, C.GoString(message))
			C.free(unsafe.Pointer(message))
			return nil, err
		}
		plugin := &Plugin{obj: obj}
		runtime.AddCleanup(plugin, core.Release, unsafe.Pointer(obj))
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

// MenuItems returns owned wrappers. The plugin is responsible for keeping its
// action targets alive while the items are installed.
func (plugin *Plugin) MenuItems() []*appkit.MenuItem {
	defer runtime.KeepAlive(plugin)
	values := C.pacman_plugin_menu_items(plugin.obj)
	defer C.free(unsafe.Pointer(values.items))
	items := make([]*appkit.MenuItem, values.count)
	for index, obj := range unsafe.Slice(values.items, values.count) {
		items[index] = appkit.MenuItemFromPointer(unsafe.Pointer(obj))
	}
	return items
}
