package config_test

import (
	"github.com/dalibo/ldap2pg/internal/config"
	"github.com/lithammer/dedent"
	"gopkg.in/yaml.v3"
)

// test helper to build a config object from a YAML string.
//
// rawYAML MUST BE NORMALIZED. No alias, no single entries, etc.
func configFromYAML(rawYAML string) (c config.Config) {
	rawYAML = dedent.Dedent(rawYAML)
	var out interface{}
	_ = yaml.Unmarshal([]byte(rawYAML), &out)
	_ = config.DecodeYaml(out, &c)
	return
}

func (suite *Suite) TestSyncItemStatic() {
	r := suite.Require()

	c := configFromYAML(`
	sync_map:
	- roles:
	  - name: "toto"
	`)
	i := c.SyncItems[0]
	i.InferAttributes()
	r.False(i.HasLDAPSearch())
	r.False(i.HasSubsearch())
}

func (suite *Suite) TestSyncItemLdapAnalyze() {
	r := suite.Require()

	c := configFromYAML(`
	sync_map:
	- ldapsearch:
	    base: cn=toto
	  roles:
	  - name: "{member.SAMAccountName}"
	`)
	i := c.SyncItems[0]
	i.InferAttributes()
	r.True(i.HasLDAPSearch())
	r.True(i.HasSubsearch())
	r.Equal("member", i.LdapSearch.SubsearchAttribute())
}
