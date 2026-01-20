package menuet

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

#include <stdlib.h>

#import "alert.h"

*/
import "C"

import (
	"unsafe"

	"github.com/gilliginsisland/pacman/pkg/syncutil"
)

var alerts = syncutil.Map[uintptr, chan<- AlertResponse]{}

// Alert represents an NSAlert
type Alert struct {
	MessageText     string
	InformativeText string
	Buttons         []string
	Inputs          []string
}

// AlertClicked represents a selected alert button
type AlertResponse struct {
	Button int
	Inputs []string
}

// Alert shows an alert, and returns the index of the button pressed, or -1 if none
func Display(alert Alert) AlertResponse {
	calert := C.make_alert()
	defer C.destroy_alert(calert)
	*calert = C.Alert{
		messageText:     C.CString(alert.MessageText),
		informativeText: C.CString(alert.InformativeText),
		buttons:         toAlertNode(alert.Buttons),
		inputs:          toAlertNode(alert.Inputs),
	}
	ch := make(chan AlertResponse, 1)
	alerts.Store(uintptr(unsafe.Pointer(calert)), ch)
	C.show_alert(calert)
	return <-ch
}

//export go_alert_clicked
func go_alert_clicked(calert *C.Alert, cresp *C.AlertResponse) {
	defer C.destroy_alert_response(cresp)
	ch, ok := alerts.LoadAndDelete(uintptr(unsafe.Pointer(calert)))
	if !ok {
		return
	}
	ch <- AlertResponse{
		Button: int(cresp.button),
		Inputs: fromAlertNode(cresp.inputs),
	}
}

func toAlertNode(s []string) *C.AlertNode {
	var node *C.AlertNode
	curr := &node
	for _, label := range s {
		cstr := C.CString(label)
		*curr = C.make_alert_node(nil)
		(*curr).text = cstr
		curr = &(*curr).next
	}
	return node
}

func fromAlertNode(node *C.AlertNode) []string {
	var s []string
	for curr := node; curr != nil; curr = curr.next {
		s = append(s, C.GoString(curr.text))
	}
	return s
}
