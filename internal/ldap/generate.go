// Implement dynamic formatting from LDAP entry.
package ldap

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	ldap3 "github.com/go-ldap/ldap/v3"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

var KnownRDNs = []string{"cn", "l", "st", "o", "ou", "c", "street", "dc", "uid"}

// Holds a consistent set of entry and sub-search entries.
type Result struct {
	// Is nil for static generation
	Entry *ldap3.Entry
	// Is empty if no sub-search.
	SubsearchAttribute string
	SubsearchEntries   []*ldap3.Entry
}

func (r *Result) GenerateValues(fmts ...pyfmt.Format) <-chan map[string]string {
	expressions := pyfmt.ListExpressions(fmts...)
	attributes := pyfmt.ListVariables(expressions...)

	// If sub-search, we want to combine parent attributes with all
	// combinations of sub-entries at once. We prepare sub-entries
	// combination and index them by a string key to combine keys with
	// parent values, all string lists.
	//
	// subMap["subentry0-comb0"] = {"cn": "toto"}
	var subMap map[string]map[string]string
	if "" != r.SubsearchAttribute {
		subMap = r.GenerateSubsearchValues(expressions)
	}

	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		for values := range r.GenerateCombinations(attributes, maps.Keys(subMap)) {
			ch <- r.ResolveExpressions(expressions, values, subMap)
		}
	}()
	return ch
}

// Return a list of expression -> values for formatting, indexed by a string key.
func (r *Result) GenerateSubsearchValues(parentExpressions []string) map[string]map[string]string {
	prefix := r.SubsearchAttribute + "."
	// First, remove sub-attribute from parent expressions. For example :
	// {member.sAMAccountName} become {sAMAccountname} in the scope of the
	// sub-entry.
	var expressions []string
	for _, e := range parentExpressions {
		if strings.HasPrefix(e, prefix) {
			expressions = append(expressions, strings.TrimPrefix(e, prefix))
		}
	}
	subAttributes := pyfmt.ListVariables(expressions...)
	subMap := make(map[string]map[string]string)
	for i, subEntry := range r.SubsearchEntries {
		j := 0
		subResult := Result{Entry: subEntry}
		for values := range subResult.GenerateCombinations(subAttributes, nil) {
			subKey := fmt.Sprintf("subentry%d-comb%d", i, j)
			values = subResult.ResolveExpressions(expressions, values, nil)
			subMap[subKey] = values
			j++
		}
	}
	return subMap
}

func (r *Result) GenerateCombinations(attributes, subKeys []string) <-chan map[string]string {
	// Extract raw LDAP attributes values from entry.
	valuesList := make([][]string, len(attributes))
	for i, attr := range attributes {
		lowerAttr := strings.ToLower(attr)
		if "dn" == attr {
			valuesList[i] = []string{r.Entry.DN}
		} else if slices.Contains(KnownRDNs, lowerAttr) {
			value0, err := ResolveFirstRDN(r.Entry.DN, attr)
			if err != nil {
				slog.Warn("Failed to read value from DN.", "dn", r.Entry.DN, "rdn", attr, "err", err)
			}
			valuesList[i] = []string{value0}
		} else if r.SubsearchAttribute == attr {
			valuesList[i] = subKeys
		} else {
			valuesList[i] = r.Entry.GetAttributeValues(attr)
		}
	}

	ch := make(chan map[string]string)
	go func() {
		defer close(ch)
		// Generate cartesian product of values and returns a map ready for
		// formatting.
		for item := range lists.Product(valuesList...) {
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

// Resolve format expression from entry or pre-resolved expression for sub-entries.
func (r *Result) ResolveExpressions(expressions []string, attrValues map[string]string, subExprMap map[string]map[string]string) map[string]string {
	exprMap := make(map[string]string)
	for _, expr := range expressions {
		attr, field, hasField := strings.Cut(expr, ".")
		if !hasField {
			// Case: {member}
			exprMap[expr] = attrValues[attr]
			continue
		}

		// Case {member.sAMAccountName}
		if attr == r.SubsearchAttribute {
			exprMap[expr] = subExprMap[attrValues[attr]][field]
			continue
		}

		// Case {member.cn}
		dn := attrValues[attr]
		value0, err := ResolveFirstRDN(dn, field)
		if err != nil {
			slog.Warn("Failed to resolve expression.", "attribute", attr, "dn", dn, "rdn", field, "err", err)
			continue
		}
		exprMap[expr] = value0
	}
	return exprMap
}

func ResolveFirstRDN(rawDN, relativeField string) (string, error) {
	dn, err := ldap3.ParseDN(rawDN)
	if err != nil {
		return "", err
	}

	for _, rdn := range dn.RDNs {
		attr0 := rdn.Attributes[0]
		if !strings.EqualFold(relativeField, attr0.Type) {
			continue
		}
		return attr0.Value, nil
	}

	return "", fmt.Errorf("no such RDN in DN")
}
