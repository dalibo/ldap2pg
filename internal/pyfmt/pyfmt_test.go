package pyfmt_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slog"
)

// Declare a global suite for the package.
type Suite struct {
	suite.Suite
}

func Test(t *testing.T) {
	if testing.Verbose() {
		internal.SetLoggingHandler(slog.LevelDebug, false)
	} else {
		internal.SetLoggingHandler(slog.LevelWarn, false)
	}
	suite.Run(t, new(Suite))
}
