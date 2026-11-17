package corefoundation

/*
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"

// Borrowed CFBoolean constants; no retain or release is required.
var (
	BooleanTrue  = TypeRef(C.kCFBooleanTrue)
	BooleanFalse = TypeRef(C.kCFBooleanFalse)
)
