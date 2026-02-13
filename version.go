package refract

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var version string

func Version() string {
	return strings.TrimSpace(version)
}

// embedded by ldflags, see goreleaser.yaml
var BuildCommit string
var BuildDate string
