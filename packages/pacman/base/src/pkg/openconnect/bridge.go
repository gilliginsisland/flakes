package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

//export go_validate_peer_cert
func go_validate_peer_cert(context unsafe.Pointer, cert *C.char) C.int {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.ValidatePeerCert == nil {
		return C.int(1)
	}
	if v.ValidatePeerCert(C.GoString(cert)) {
		return C.int(0)
	}
	return C.int(1)
}

//export go_process_auth_form
func go_process_auth_form(context unsafe.Pointer, form *C.struct_oc_auth_form) C.int {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.ProcessAuthForm == nil {
		return C.OC_FORM_RESULT_ERR
	}

	// Read form metadata
	f := AuthForm{
		Banner:  C.GoString(form.banner),
		Message: C.GoString(form.message),
		Error:   C.GoString(form.error),
		Options: []FormOption{},
	}

	// Process authentication group selection
	if opt := form.authgroup_opt; opt != nil {
		choices := make([]FormChoice, opt.nr_choices)
		for i, choice := range unsafe.Slice(opt.choices, opt.nr_choices) {
			choices[i] = FormChoice{
				Name:  C.GoString(choice.name),
				Label: C.GoString(choice.label),
			}
		}

		f.AuthGroup = &FormOption{
			Name:    C.GoString(opt.form.name),
			Label:   C.GoString(opt.form.label),
			Choices: choices,
		}
	}

	// Process form fields
	for opt := form.opts; opt != nil; opt = opt.next {
		if opt.flags&C.OC_FORM_OPT_IGNORE != 0 {
			continue
		}

		option := FormOption{
			handle: opt,
			Name:   C.GoString(opt.name),
			Label:  C.GoString(opt.label),
			Type:   FormOptionType(opt._type),
		}

		if opt._type == C.OC_FORM_OPT_SELECT {
			opt := (*C.struct_oc_form_opt_select)(unsafe.Pointer(opt))

			choices := make([]FormChoice, opt.nr_choices)
			for i, choice := range unsafe.Slice(opt.choices, opt.nr_choices) {
				choices[i] = FormChoice{
					Name:  C.GoString(choice.name),
					Label: C.GoString(choice.label),
				}
			}
			option.Choices = choices
		}

		f.Options = append(f.Options, option)
	}

	// Pass converted form to the callback
	return C.int(v.ProcessAuthForm(&f))
}

//export go_process_form_error
func go_process_form_error(context unsafe.Pointer, message *C.char) {
	v, ok := handles.Load(uintptr(context))
	if !ok {
		return
	}

	select {
	case v.errCh <- &OpError{Op: "auth", Err: errors.New(C.GoString(message))}:
	default:
	}
}

//export go_process_csd
func go_process_csd(context unsafe.Pointer, hostname, sha256, token, ticket, stub *C.char) C.int {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.ProcessCSD == nil {
		return C.int(-22)
	}

	info := CSDInfo{
		Hostname: goString(hostname),
		SHA256:   goString(sha256),
		Token:    goString(token),
		Ticket:   goString(ticket),
		Stub:     goString(stub),
	}

	err := v.ProcessCSD(info)
	if err == nil {
		return 0
	}

	select {
	case v.errCh <- &OpError{Op: "hostscan", Err: err}:
	default:
	}
	return C.int(-22)
}

func goString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

//export go_progress
func go_progress(context unsafe.Pointer, level C.int, message *C.char) {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.Progress == nil {
		return
	}
	v.Progress(LogLevel(level), C.GoString(message))
}

//export go_external_browser_callback
func go_external_browser_callback(_ *C.struct_openconnect_info, uri *C.char, context unsafe.Pointer) C.int {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.ExternalBrowser == nil {
		return 1
	}

	if err := v.ExternalBrowser(C.GoString(uri)); err != nil {
		return 1
	}

	return 0
}

//export go_reconnected_handler
func go_reconnected_handler(context unsafe.Pointer) {
	v, ok := handles.Load(uintptr(context))
	if !ok || v.ReconnectedHandler == nil {
		return
	}
	v.ReconnectedHandler()
}

//export go_mainloop_result
func go_mainloop_result(context unsafe.Pointer, result C.int) {
	v, ok := handles.Load(uintptr(context))
	if !ok {
		return
	}

	err := ocErrno("main loop", result)
	if err == nil {
		return
	}

	select {
	case v.errCh <- err:
	default:
	}
}
