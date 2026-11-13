package core

/*
#include <stdlib.h>

#include "core.h"
*/
import "C"

import "unsafe"

type SelectorRef = *C.struct_objc_selector

type Selector struct {
	sel C.SEL
}

func SEL(name string) Selector {
	value := C.CString(name)
	defer C.free(unsafe.Pointer(value))
	return Selector{sel: C.menuet_selector(value)}
}

func (selector Selector) Pointer() SelectorRef {
	return (*C.struct_objc_selector)(selector.sel)
}
