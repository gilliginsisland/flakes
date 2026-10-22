package core

/*
#include <stdlib.h>

#include "core.h"
*/
import "C"

import "unsafe"

type Selector struct {
	sel C.SEL
}

func SEL(name string) Selector {
	value := C.CString(name)
	defer C.free(unsafe.Pointer(value))
	return Selector{sel: C.menuet_selector(value)}
}

func (selector Selector) Pointer() C.SEL {
	return selector.sel
}
