package dialer

//go:generate go tool stringer -type=ConnectionState
type ConnectionState int

const (
	Offline ConnectionState = iota
	Connecting
	Failed
	Online
)

type StateUpdate struct {
	State ConnectionState
	Err   error
}
