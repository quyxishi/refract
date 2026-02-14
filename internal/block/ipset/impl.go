package ipset

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
	"github.com/quyxishi/refract/internal/block"
	"github.com/quyxishi/refract/internal/serial"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

const IPSET_NAME string = "refract_banned_users"

type IpsetBlockStrategy struct {
	Timeout         uint32
	Protocol        serial.TransportProto
	DestinationPort uint16
}

func (h *IpsetBlockStrategy) Name() string {
	return "Ipset"
}

func (h *IpsetBlockStrategy) Init() error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to construct iptables instance due: %v", err)
	}

	err = netlink.IpsetCreate(IPSET_NAME, "hash:ip", netlink.IpsetCreateOptions{Timeout: &h.Timeout, Replace: true})
	if err != nil && err.Error() != "file exists" {
		return fmt.Errorf("failed to create ipset:%s due: %v", IPSET_NAME, err)
	}

	rule := []string{
		"-p", strings.ToLower(string(h.Protocol)), "--dport", strconv.FormatUint(uint64(h.DestinationPort), 10),
		"-m", "set", "--match-set", IPSET_NAME, "src",
		"-j", "DROP",
	}

	exists, err := ipt.Exists("filter", "INPUT", rule...)
	if err != nil {
		return fmt.Errorf("failed to check iptables rule existence: %v", err)
	}

	if !exists {
		err = ipt.Insert("filter", "INPUT", 1, rule...)
		if err != nil {
			return fmt.Errorf("failed to insert rule in iptables due: %v", err)
		}
	}

	return nil
}

func (h *IpsetBlockStrategy) Block(ip net.IP) error {
	// Ignore "already exists" errors since netlink doesn't export a "ErrExists" constant for ipset,
	// but functionally we just proceed if it fails.
	var _ = netlink.IpsetAdd(IPSET_NAME, &netlink.IPSetEntry{IP: ip})

	killSockets := func(family uint8) error {
		switch h.Protocol {
		case serial.ProtoTCP:
			sockets, err := netlink.SocketDiagTCPInfo(family)
			if err != nil {
				return fmt.Errorf("failed to dump tcp sockets via netlink due: %v", err)
			}

			for _, sock := range sockets {
				// Filter for established connections only
				if sock.InetDiagMsg.State != nl.TCP_CONNTRACK_ESTABLISHED {
					continue
				}

				// Filter to match our destination port
				if sock.InetDiagMsg.ID.SourcePort != h.DestinationPort {
					continue
				}

				// Check if the Destination IP (remote user) matches our target
				if sock.InetDiagMsg.ID.Destination.Equal(ip) {
					localAddr := &net.TCPAddr{
						IP:   sock.InetDiagMsg.ID.Source,
						Port: int(sock.InetDiagMsg.ID.SourcePort),
					}
					remoteAddr := &net.TCPAddr{
						IP:   sock.InetDiagMsg.ID.Destination,
						Port: int(sock.InetDiagMsg.ID.DestinationPort),
					}

					// Destroy the specific socket identified by the message
					var _ = netlink.SocketDestroy(localAddr, remoteAddr)
				}
			}
		case serial.ProtoUDP:
			filter := &netlink.ConntrackFilter{}

			// Filter by Layer 3 (IP) Protocol
			var _ = filter.AddIP(netlink.ConntrackOrigSrcIP, ip)
			// Filter by Layer 4 (Transport) Protocol
			var _ = filter.AddProtocol(syscall.IPPROTO_UDP)
			// Filter by Destination Port
			var _ = filter.AddPort(netlink.ConntrackOrigDstPort, uint16(h.DestinationPort))

			// Execute deletion from Conntrack table
			var _, _ = netlink.ConntrackDeleteFilters(netlink.ConntrackTable, netlink.InetFamily(family), filter)
		default:
			return fmt.Errorf("unsupported protocol: %s", h.Protocol)
		}

		return nil
	}

	// Kill IPv4
	if err := killSockets(syscall.AF_INET); err != nil {
		return err
	}

	// Kill IPv6 (if needed)
	if ip.To4() == nil {
		if err := killSockets(syscall.AF_INET6); err != nil {
			return err
		}
	}

	return nil
}

// Compile-time assertion to ensure that strategy satisfies interface
var _ block.BlockStrategy = (*IpsetBlockStrategy)(nil)
