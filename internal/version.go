package internal

import (
	_ "embed"

	"github.com/carlmjohnson/versioninfo"
)

var (
	//go:embed VERSION
	Version       string
	ShortRevision string
)

func init() {
	if 8 > len(versioninfo.Revision) {
		ShortRevision = "(unknown)"
	} else {
		ShortRevision = versioninfo.Revision[:8]
	}
}
