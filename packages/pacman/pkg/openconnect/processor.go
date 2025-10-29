package openconnect

import (
	"log/slog"
	"strings"
)

var (
	_ FormProcessor = (*CredentialsProcessor)(nil)
	_ FormProcessor = (*LoggerFunc)(nil)
	_ FormProcessor = (*AggregateProcessor)(nil)
	_ FormProcessor = (*FormProcessorFn)(nil)
)

// FormProcessor defines an interface for processing authentication forms.
type FormProcessor interface {
	ProcessForm(form *AuthForm) FormResult
}

// FormProcessorFn is a function type that implements the FormProcessor interface.
type FormProcessorFn func(form *AuthForm) FormResult

// ProcessForm calls the function itself to satisfy the FormProcessor interface.
func (fn FormProcessorFn) ProcessForm(form *AuthForm) FormResult {
	return fn(form)
}

// CredentialsProcessor handles setting username and password fields in the form.
type CredentialsProcessor struct {
	Username string
	Password string
}

// ProcessForm implements the FormProcessor interface to set username and password fields.
func (cp *CredentialsProcessor) ProcessForm(form *AuthForm) FormResult {
	for _, opt := range form.Options {
		switch opt.Type {
		case FormOptionText:
			if strings.HasPrefix(strings.ToLower(opt.Name), "user") {
				opt.SetValue(cp.Username)
			}
		case FormOptionPassword:
			opt.SetValue(cp.Password)
		}
	}
	return FormResultOk
}

// LoggerFunc defines a function type for logging messages with attributes.
type LoggerFunc func(msg string, attrs ...slog.Attr)

// ProcessForm implements the FormProcessor interface to log form details.
func (lf LoggerFunc) ProcessForm(form *AuthForm) FormResult {
	lf("Processing Auth Form",
		slog.String("banner", form.Banner),
		slog.String("message", form.Message),
		slog.String("error", form.Error),
	)

	for _, opt := range form.Options {
		lf("option",
			slog.String("name", opt.Name),
			slog.String("label", opt.Label),
			slog.String("type", opt.Type.String()),
		)
		for _, choice := range opt.Choices {
			lf("choice",
				slog.String("name", choice.Name),
				slog.String("label", choice.Label),
			)
		}
	}
	return FormResultOk
}

// AggregateProcessor combines multiple FormProcessors and calls them in sequence.
type AggregateProcessor []FormProcessor

// ProcessForm calls each processor in sequence, returning the first error encountered.
func (ap AggregateProcessor) ProcessForm(form *AuthForm) FormResult {
	for _, processor := range ap {
		if result := processor.ProcessForm(form); result != FormResultOk {
			return result
		}
	}
	return FormResultOk
}
