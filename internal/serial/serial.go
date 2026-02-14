package serial

import (
	"net"
	"time"
)

type TransportProto string

const (
	ProtoTCP TransportProto = "tcp"
	ProtoUDP TransportProto = "udp"
)

// UserState tracks the last valid connection info for a user
type UserState struct {
	LastIP   net.IP
	LastSeen time.Time
}

type LogEntry struct {
	IP    net.IP
	Email string
	Tag   string
}
