package ldap

import (
	"fmt"

	ldap3 "github.com/go-ldap/ldap/v3"
)

type Scope int

func ParseScope(s string) (Scope, error) {
	switch s {
	case "sub":
		return ldap3.ScopeWholeSubtree, nil
	case "base":
		return ldap3.ScopeBaseObject, nil
	case "one":
		return ldap3.ScopeSingleLevel, nil
	default:
		return 0, fmt.Errorf("bad scope: %s", s)
	}
}

func (s Scope) String() string {
	switch s {
	case ldap3.ScopeBaseObject:
		return "base"
	case ldap3.ScopeWholeSubtree:
		return "sub"
	case ldap3.ScopeSingleLevel:
		return "one"
	default:
		return "!INVALID"
	}
}
