package systemconfiguration

/*
#include <SystemConfiguration/SystemConfiguration.h>
#include <dispatch/dispatch.h>

void menuet_store_changed(SCDynamicStoreRef store, CFArrayRef keys, void *info);
*/
import "C"

import (
	"runtime"
	"unsafe"

	cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"
	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

// DynamicStore owns a native dynamic-store session.
type DynamicStore struct {
	ref      C.SCDynamicStoreRef
	callback func()
}

// Scheduled stores stay alive until notification delivery is explicitly stopped.
var storeCallbacks syncutil.Map[C.SCDynamicStoreRef, *DynamicStore]

// NewDynamicStore creates a session with a fixed notification handler.
// The handler runs in a goroutine; notification keys and scheduling are separate.
func NewDynamicStore(name string, callback func()) (*DynamicStore, error) {
	text := cf.NewString(name)
	defer C.CFRelease(C.CFTypeRef(text))
	ref := C.SCDynamicStoreCreate(0, C.CFStringRef(text), C.SCDynamicStoreCallBack(C.menuet_store_changed), nil)
	if ref == 0 {
		return nil, configurationError("create dynamic store")
	}
	store := &DynamicStore{ref: ref, callback: callback}
	runtime.AddCleanup(store, func(ref C.SCDynamicStoreRef) {
		C.CFRelease(C.CFTypeRef(ref))
	}, ref)
	return store, nil
}

func (store *DynamicStore) CopyProxies() (*cf.Dictionary, error) {
	defer runtime.KeepAlive(store)
	value := cf.OwnDictionary(cf.DictionaryRef(C.SCDynamicStoreCopyProxies(store.ref)))
	if value == nil {
		return nil, configurationError("read effective proxy settings")
	}
	return value, nil
}

// CreateProxiesKey returns an owned CF string. The caller must release it.
func CreateProxiesKey() cf.StringRef {
	return cf.StringRef(C.SCDynamicStoreKeyCreateProxies(0))
}

func (store *DynamicStore) SetNotificationKeys(keys ...cf.StringRef) error {
	defer runtime.KeepAlive(store)
	refs := cf.NewArray(keys)
	defer cf.Release(cf.TypeRef(refs))
	if C.SCDynamicStoreSetNotificationKeys(store.ref, C.CFArrayRef(refs), 0) == 0 {
		return configurationError("set dynamic-store notification keys")
	}
	return nil
}

// Schedule starts notifications and keeps the store alive until Stop is called.
// Calls to Schedule and Stop on the same store must be serialized by the caller.
func (store *DynamicStore) Schedule() error {
	defer runtime.KeepAlive(store)
	if _, loaded := storeCallbacks.LoadOrStore(store.ref, store); loaded {
		return nil
	}
	if C.SCDynamicStoreSetDispatchQueue(store.ref, C.dispatch_get_global_queue(C.QOS_CLASS_DEFAULT, 0)) == 0 {
		err := configurationError("schedule dynamic-store notifications")
		storeCallbacks.CompareAndDelete(store.ref, store)
		return err
	}
	return nil
}

// Stop unregisters the handler and disables native notifications. It does not
// release the session. Handlers already dispatched may still finish running.
func (store *DynamicStore) Stop() error {
	defer runtime.KeepAlive(store)
	if !storeCallbacks.CompareAndDelete(store.ref, store) {
		return nil
	}
	if C.SCDynamicStoreSetDispatchQueue(store.ref, nil) == 0 {
		err := configurationError("stop dynamic-store notifications")
		storeCallbacks.Store(store.ref, store)
		return err
	}
	return nil
}

//export menuet_store_changed
func menuet_store_changed(ref C.SCDynamicStoreRef, keys C.CFArrayRef, info unsafe.Pointer) {
	store, ok := storeCallbacks.Load(ref)
	if ok && store.callback != nil {
		go store.callback()
	}
}
