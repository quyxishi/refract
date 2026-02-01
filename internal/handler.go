package internal

import (
	"log"
	"time"

	"github.com/quyxishi/refract/internal/block"
	"github.com/quyxishi/refract/internal/serial"
)

type LogsHandler struct {
	Events   <-chan serial.LogEntry
	Window   time.Duration
	Timeout  time.Duration
	Strategy block.BlockStrategy
}

func (h *LogsHandler) Serve() {
	// State for user sessions
	state := make(map[string]serial.UserState)

	// Cache for recently blocked IPs to avoid spamming the kernel
	blockedCache := make(map[string]time.Time)

	debounceDuration := h.Timeout
	if debounceDuration < time.Second {
		debounceDuration = time.Second
	}

	// Ticker to clean up the blocked cache periodically
	cleanupTicker := time.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-cleanupTicker.C:
			now := time.Now()
			for ip, timestamp := range blockedCache {
				if now.Sub(timestamp) > debounceDuration {
					delete(blockedCache, ip)
				}
			}
		case entry, ok := <-h.Events:
			if !ok {
				return // Channel closed
			}

			user, exists := state[entry.Email]

			// If new user or window expired, update state to this new IP
			if !exists || time.Since(user.LastSeen) > h.Window {
				state[entry.Email] = serial.UserState{
					LastIP:   entry.IP,
					LastSeen: time.Now(),
				}
				continue
			}

			// User exists and is within window, check for violation
			if !user.LastIP.Equal(entry.IP) {
				ipStr := entry.IP.String()
				lastBlockTime, blockedRecently := blockedCache[ipStr]

				if !blockedRecently || time.Since(lastBlockTime) > debounceDuration {
					log.Printf("disallowed concurrent connection from %s w/ origin %s [%s] email: %s\n",
						entry.IP,
						user.LastIP,
						entry.Tag,
						entry.Email,
					)

					var _ = h.Strategy.Block(entry.IP)
					blockedCache[ipStr] = time.Now()
				}

			} else {
				// Same IP, just update the timestamp to keep the session alive
				user.LastSeen = time.Now()
				state[entry.Email] = user
			}
		}

	}
}
