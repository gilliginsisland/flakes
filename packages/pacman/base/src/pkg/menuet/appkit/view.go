package appkit

/*
#import <Cocoa/Cocoa.h>
*/
import "C"

type Rect struct {
	X, Y, Width, Height float64
}

type AutoresizingMask uint64

const (
	ViewWidthSizable  AutoresizingMask = C.NSViewWidthSizable
	ViewHeightSizable AutoresizingMask = C.NSViewHeightSizable
)
