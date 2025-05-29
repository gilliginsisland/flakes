package prompt

// InputType defines the type of input expected in a Dialog.
type InputType int

const (
	NoInput InputType = iota
	TextInput
	SecureInput
)

// Dialog defines the user input prompt settings.
type Dialog struct {
	Message       string
	Title         string
	Input         InputType
	DefaultAnswer string
	Buttons       []string
	DefaultButton string
	CancelButton  string
}

// Prompter defines the interface to show input dialogs.
type Prompter interface {
	Prompt(d Dialog) (string, error)
}

// Prompt invokes the platform-specific implementation.
func Prompt(d Dialog) (string, error) {
	return prompter{}.Prompt(d)
}
