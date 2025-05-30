package ldap_test

import (
	"github.com/dalibo/ldap2pg/v6/internal/ldap"
	ldap3 "github.com/go-ldap/ldap/v3"
)

func (suite *Suite) TestCleanFilter() {
	r := suite.Require()
	var (
		f   string
		err error
	)

	f = ldap.CleanFilter("(cn=dba)")
	_, err = ldap3.CompileFilter(f)
	r.Nil(err, f)

	f = ldap.CleanFilter("  (& (cn=dba) (member=*) ) ")
	_, err = ldap3.CompileFilter(f)
	r.Nil(err, f)

	f = ldap.CleanFilter(`\n  (&\n (cn=dba)\n (member=*)\n )\n`)
	_, err = ldap3.CompileFilter(f)
	r.Nil(err, f)
}
