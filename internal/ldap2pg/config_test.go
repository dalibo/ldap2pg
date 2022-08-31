package ldap2pg_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/ldap2pg"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestLoadEnvDoesntOverwrite(t *testing.T) {
	ldap2pg.Logger = zaptest.NewLogger(t).Sugar()
	defer func() {
		ldap2pg.Logger = nil
	}()

	r := require.New(t)

	config := ldap2pg.Config{
		ConfigFile: "defined-ldap2pg.yaml",
	}
	values := ldap2pg.EnvValues{
		ConfigFile: "env-ldap2pg.yaml",
	}
	config.LoadEnv(values)

	r.Equal(config.ConfigFile, "defined-ldap2pg.yaml")
}
