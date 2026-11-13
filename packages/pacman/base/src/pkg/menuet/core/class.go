package core

/*
#include "core.h"
*/
import "C"

// ClassRef is a borrowed Objective-C runtime class handle.
type ClassRef = *C.struct_objc_class
