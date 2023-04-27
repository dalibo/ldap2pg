// Implement dynamic formatting from LDAP entry.
package ldap

import (
	"strings"

	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"github.com/dalibo/ldap2pg/internal/utils"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/slog"
)

// Generate all combination of attribute values from entry as referenced by formats in fmts.
func GenerateValues(entry *ldap3.Entry, fmts ...pyfmt.Format) <-chan map[string]string {
	expressions := pyfmt.ListExpressions(fmts...)
	attributes := pyfmt.ListVariables(expressions...)
	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		for values := range GenerateCombinations(entry, attributes) {
			ch <- ResolveExpressions(expressions, values)
		}
	}()
	return ch
}

func GenerateCombinations(entry *ldap3.Entry, attributes []string) <-chan map[string]string {
	// Extract raw LDAP attributes values from entry.
	valuesList := make([][]string, len(attributes))
	for i, attr := range attributes {
		if "dn" == attr {
			valuesList[i] = []string{entry.DN}
		} else {
			valuesList[i] = entry.GetAttributeValues(attr)
		}
	}

	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		// Generate cartesian product of values and returns a map ready for
		// formatting.
		for item := range utils.Product(valuesList...) {
			// Index values by attributes.
			attrMap := make(map[string]string)
			for i, attr := range attributes {
				attrMap[attr] = item[i]
			}
			ch <- attrMap
		}
	}()
	return ch
}

// Map expresssion to the corresponding value from attributes.
func ResolveExpressions(expressions []string, attrValues map[string]string) map[string]string {
	exprMap := make(map[string]string)
exprloop:
	for _, expr := range expressions {
		attr, field, hasField := strings.Cut(expr, ".")
		if !hasField {
			// Case: {member}
			exprMap[expr] = attrValues[attr]
			continue
		}

		// Case {member.cn}
		dn, _ := ldap3.ParseDN(attrValues[attr])
		for _, rdn := range dn.RDNs {
			attr0 := rdn.Attributes[0]
			if field == attr0.Type {
				exprMap[expr] = attr0.Value
				continue exprloop
			}
		}

		slog.Warn("Unexpected DN.", "dn", dn, "rdn", field)
	}
	return exprMap
}
