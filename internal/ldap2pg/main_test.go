// Global unit test suite.
package ldap2pg_test

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func TestLdap2pg(t *testing.T) {
	if testing.Verbose() {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.PanicLevel)
	}
	suite.Run(t, new(TestSuite))
	suite.Run(t, new(ConfigSuite))
}
