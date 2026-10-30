package dialer

//go:generate go tool stringer -type=ConnectionState
type ConnectionState int

const (
	Idle ConnectionState = iota
	Connecting
	Failed
	Online
)

type StateUpdate struct {
	State ConnectionState
	Err   error
}
