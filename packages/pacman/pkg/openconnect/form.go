package openconnect

/*
#cgo pkg-config: openconnect
#include <openconnect.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"unsafe"
)

//go:generate go run ./vendor/golang.org/x/tools/cmd/stringer -type=FormOptionType -trimprefix=FormOption
type FormOptionType int

const (
	FormOptionText     FormOptionType = C.OC_FORM_OPT_TEXT
	FormOptionPassword FormOptionType = C.OC_FORM_OPT_PASSWORD
	FormOptionSelect   FormOptionType = C.OC_FORM_OPT_SELECT
	FormOptionHidden   FormOptionType = C.OC_FORM_OPT_HIDDEN
	FormOptionToken    FormOptionType = C.OC_FORM_OPT_TOKEN
	FormOptionSSOToken FormOptionType = C.OC_FORM_OPT_SSO_TOKEN
	FormOptionSSOUser  FormOptionType = C.OC_FORM_OPT_SSO_USER
)

type FormChoice struct {
	Name  string
	Label string
}

type FormOption struct {
	handle  *C.struct_oc_form_opt
	Name    string
	Label   string
	Type    FormOptionType
	Choices []FormChoice
}

func (o *FormOption) SetValue(val string) error {
	cStr := C.CString(val)
	defer C.free(unsafe.Pointer(cStr))
	if C.openconnect_set_option_value(o.handle, cStr) != 0 {
		return errors.New("failed to set option value")
	}
	return nil
}

type AuthForm struct {
	Banner    string
	Message   string
	Error     string
	AuthGroup *FormOption
	Options   []FormOption
}

//go:generate go run ./vendor/golang.org/x/tools/cmd/stringer -type=FormResult -trimprefix=FormResult
type FormResult int

const (
	FormResultErr       FormResult = C.OC_FORM_RESULT_ERR
	FormResultOk        FormResult = C.OC_FORM_RESULT_OK
	FormResultCancelled FormResult = C.OC_FORM_RESULT_CANCELLED
	FormResultNewgroup  FormResult = C.OC_FORM_RESULT_NEWGROUP
)
