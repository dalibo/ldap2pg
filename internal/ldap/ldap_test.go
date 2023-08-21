package ldap_test

import (
	"log/slog"
	"testing"

	"github.com/dalibo/ldap2pg/internal"
	"github.com/stretchr/testify/suite"
)

// Global test suite for ldap package.
type Suite struct {
	suite.Suite
}

func TestConfig(t *testing.T) {
	if testing.Verbose() {
		internal.SetLoggingHandler(slog.LevelDebug, false)
	} else {
		internal.SetLoggingHandler(slog.LevelWarn, false)
	}
	suite.Run(t, new(Suite))
}
