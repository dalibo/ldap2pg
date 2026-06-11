package homedir_test

import (
	"os"
	"strings"
	"testing"

	"github.com/dalibo/ldap2pg/v6/internal/homedir"
	"github.com/stretchr/testify/require"
)

func TestNotHome(t *testing.T) {
	r := require.New(t)

	r.Equal("pif~", homedir.Expand("pif~"))
}

func TestBareTilde(t *testing.T) {
	r := require.New(t)

	h, _ := os.UserHomeDir()
	r.Equal(h, homedir.Expand("~"))
}

func TestCurrentHome(t *testing.T) {
	r := require.New(t)

	h, _ := os.UserHomeDir()
	r.True(strings.HasPrefix(homedir.Expand("~/ldap2pg.yml"), h))
}

func TestOtherHome(t *testing.T) {
	r := require.New(t)

	r.Equal("/root/ldap2pg.yml", homedir.Expand("~root/ldap2pg.yml"))
}

func TestBareTildeOtherHome(t *testing.T) {
	r := require.New(t)

	r.Equal("/root", homedir.Expand("~root"))
}
