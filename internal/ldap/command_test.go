package ldap_test

import "github.com/dalibo/ldap2pg/internal/ldap"

func (suite *Suite) TestCommandSearch() {
	r := suite.Require()

	c := ldap.Client{
		URI: "ldaps://pouet",
	}
	cmd := c.Command("ldapsearch", "(filter=*)", "cn", "member")
	r.Equal(`ldapsearch -H ldaps://pouet -x '(filter=*)' cn member`, cmd)
}

func (suite *Suite) TestQuote() {
	r := suite.Require()

	r.Equal(`''`, ldap.ShellQuote(""))
	r.Equal(`"'"`, ldap.ShellQuote("'"))
	r.Equal(`'"'`, ldap.ShellQuote(`"`))
	r.Equal(`' '`, ldap.ShellQuote(` `))
	r.Equal("'`'", ldap.ShellQuote("`"))
	r.Equal(`'*'`, ldap.ShellQuote(`*`))
	r.Equal(`'!'`, ldap.ShellQuote(`!`))
	r.Equal(`'(cn=*)'`, ldap.ShellQuote(`(cn=*)`))
	r.Equal(`d"'"accord`, ldap.ShellQuote(`d'accord`))
	r.Equal(`'(cn="toto")'`, ldap.ShellQuote(`(cn="toto")`))
	r.Equal(`'(cn='"'"toto"'"')'`, ldap.ShellQuote(`(cn='toto')`))
	r.Equal(`'"'"'"'"'`, ldap.ShellQuote(`"'"`))
}
