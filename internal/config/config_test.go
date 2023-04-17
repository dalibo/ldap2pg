package config_test

import (
	"testing"

	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slog"
)

type ConfigSuite struct {
	suite.Suite
}

func (suite *ConfigSuite) TestLoadEnvDoesNotOverwriteConfigFile() {
	r := suite.Require()

	c := config.Config{
		ConfigFile: "defined-ldap2pg.yaml",
	}
	values := config.EnvValues{
		ConfigFile: "env-ldap2pg.yaml",
	}
	c.LoadEnv(values)

	r.Equal(c.ConfigFile, "defined-ldap2pg.yaml")
}

func TestConfig(t *testing.T) {
	if testing.Verbose() {
		config.SetLoggingHandler(slog.LevelDebug, false)
	} else {
		config.SetLoggingHandler(slog.LevelDebug, false)
	}
	suite.Run(t, new(ConfigSuite))
}
