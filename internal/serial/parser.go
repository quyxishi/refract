package serial

import (
	"net"
	"strings"
)

// Extract fields from standard Xray access log format:
//
// 2026/01/30 17:03:38.067799 from 172.19.0.3:58676 accepted tcp:www.google.com:443 [NIDX00-INBOUND-IDX00 >> direct] email: 2
func ParseLine(line string) (LogEntry, bool) {
	// Quick filter
	if !strings.Contains(line, "accepted") {
		return LogEntry{}, false
	}

	parts := strings.Fields(line)
	if len(parts) < 10 {
		return LogEntry{}, false
	}

	// Parse Email (email: 2)
	email := parts[len(parts)-1]

	// Parse IP (from 172.19.0.3:58676)
	ipPort := parts[3]
	ip, _, found := strings.Cut(ipPort, ":")
	if !found {
		return LogEntry{}, false
	}
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return LogEntry{}, false
	}

	// Parse Tag (NIDX00-INBOUND-IDX00)
	startBracket := strings.Index(line, "[")
	endBracket := strings.Index(line, "]")
	tag := "unknown"
	if startBracket != -1 && endBracket != -1 {
		tag = line[startBracket+1 : endBracket]
	}

	return LogEntry{
		IP:    ipAddr,
		Email: email,
		Tag:   tag,
	}, true
}
