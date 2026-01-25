package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>

*/
import "C"

type FontWeight float64

var (
	WeightUltraLight FontWeight = FontWeight(C.NSFontWeightUltraLight)
	WeightThin                  = FontWeight(C.NSFontWeightThin)
	WeightLight                 = FontWeight(C.NSFontWeightLight)
	WeightRegular               = FontWeight(C.NSFontWeightRegular)
	WeightMedium                = FontWeight(C.NSFontWeightMedium)
	WeightSemibold              = FontWeight(C.NSFontWeightSemibold)
	WeightBold                  = FontWeight(C.NSFontWeightBold)
	WeightHeavy                 = FontWeight(C.NSFontWeightHeavy)
	WeightBlack                 = FontWeight(C.NSFontWeightBlack)
)
