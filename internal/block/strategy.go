package block

import "net"

type BlockStrategy interface {
	Name() string
	Init() error
	Block(ip net.IP) error
}
