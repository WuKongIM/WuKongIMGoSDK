package wksdk

// Status represents the state of the connection.
type ConnectStatus int

const (
	DISCONNECTED = ConnectStatus(iota)
	CONNECTED
	CLOSED
	RECONNECTING
	CONNECTING
)

func (s ConnectStatus) String() string {
	switch s {
	case DISCONNECTED:
		return "DISCONNECTED"
	case CONNECTED:
		return "CONNECTED"
	case CLOSED:
		return "CLOSED"
	case RECONNECTING:
		return "RECONNECTING"
	case CONNECTING:
		return "CONNECTING"
	}
	return "unknown status"
}
