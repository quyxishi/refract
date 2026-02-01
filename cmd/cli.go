package main

import (
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Version kong.VersionFlag `short:"v" help:"Print version information and exit."`

	Protocol        string `name:"proto" help:"The transport protocol (tcp/udp) used to filter connections for concurrency enforcement." default:"tcp" enum:"tcp,udp"`
	DestinationPort uint16 `name:"dport" help:"Destination port of the connections that should be monitored." default:"443"`

	Window  time.Duration `short:"w" help:"Time window for checking concurrency." default:"5s"`
	Timeout time.Duration `short:"t" help:"Duration to enforce the ban on a conflicting IP before automatically lifting the restriction." default:"1m"`

	AccessLog string `name:"access.log" help:"Path to Xray access log." default:"/var/log/xray/access.log"`
	BlockLog  string `name:"block.log" help:"Path to Refract audit log." default:"/var/log/xray/block.log"`
}
