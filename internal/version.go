package internal

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var Version string

func init() {
	Version = strings.TrimSpace(Version)
}
