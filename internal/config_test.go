package internal_test

import (
	ldap2pg "github.com/dalibo/ldap2pg/internal"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (suite *ConfigSuite) TestLoadEnvDoesNotOverwriteConfigFile() {
	r := suite.Require()

	config := ldap2pg.Config{
		ConfigFile: "defined-ldap2pg.yaml",
	}
	values := ldap2pg.EnvValues{
		ConfigFile: "env-ldap2pg.yaml",
	}
	config.LoadEnv(values)

	r.Equal(config.ConfigFile, "defined-ldap2pg.yaml")
}
