// Global unit test suite.
package internal_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slog"

	ldap2pg "github.com/dalibo/ldap2pg/internal"
)

type TestSuite struct {
	suite.Suite
}

func TestLdap2pg(t *testing.T) {
	if testing.Verbose() {
		ldap2pg.SetLoggingHandler(slog.LevelDebug)
	} else {
		ldap2pg.SetLoggingHandler(slog.LevelDebug)
	}
	suite.Run(t, new(TestSuite))
	suite.Run(t, new(ConfigSuite))
}
