package main

import (
	"github.com/dalibo/ldap2pg/v6/internal/cmd"
)

var version string // set by goreleaser

func init() {
	cmd.Version = version
}

func main() {
	cmd.Main()
}
