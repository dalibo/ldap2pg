package ldap

import (
	"regexp"
	"strings"
)

// Prepare a YAML filter string for compilation by ldapv3.CompileFilter.
// go-ldap is stricter than openldap when implementing RFC4515 filter. No
// spaces are allowed around parenthesises.
func CleanFilter(filter string) string {
	filter = strings.ReplaceAll(filter, `\n`, "")
	re, _ := regexp.Compile(`\s+\(`)
	filter = re.ReplaceAllLiteralString(filter, "(")
	re, _ = regexp.Compile(`\)\s+`)
	filter = re.ReplaceAllLiteralString(filter, ")")
	return filter
}
