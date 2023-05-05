package ldap

import (
	"fmt"
	"strings"
)

func (c Client) Command(name string, args ...string) string {
	cmd := []string{name}
	if "" != c.URI {
		cmd = append(cmd, "-H", c.URI)
	}
	if 0 != c.Timeout && "ldapsearch" == name {
		cmd = append(cmd, "-l", fmt.Sprintf("%.0f", c.Timeout.Seconds()))
	}
	if "" != c.BindDN {
		cmd = append(cmd, "-D", c.BindDN)
	}
	if "" == c.SaslMech {
		cmd = append(cmd, "-x")
	} else {
		cmd = append(cmd, "-Y", c.SaslMech)
	}
	if "" != c.SaslAuthCID {
		cmd = append(cmd, "-U", c.SaslAuthCID)
	}
	if "" != c.Password {
		cmd = append(cmd, "-w", "$LDAPPASSWORD")
	}
	cmd = append(cmd, args...)
	for i, arg := range cmd {
		cmd[i] = ShellQuote(arg)
	}
	return strings.Join(cmd, " ")
}

var specialChars = ` "*!()[]{}` + "`"

func NeedsQuote(s string) bool {
	if "" == s {
		return true
	}
	for i := range s {
		if strings.ContainsAny(s[i:i+1], specialChars) {
			return true
		}
	}
	return false
}

func ShellQuote(arg string) string {
	if "" == arg {
		return `''`
	}

	quoteParts := strings.Split(arg, `'`)
	b := strings.Builder{}
	for i, part := range quoteParts {
		if 0 < i {
			b.WriteString(`"'"`)
		}

		if "" == part {
			continue
		}

		if NeedsQuote(part) {
			b.WriteString(`'`)
			b.WriteString(part)
			b.WriteString(`'`)

		} else {
			b.WriteString(part)
		}

	}
	return b.String()
}
