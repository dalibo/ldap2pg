package privileges

import (
	"fmt"
	"strings"

	"github.com/dalibo/ldap2pg/internal/ldap"
	"github.com/dalibo/ldap2pg/internal/lists"
	"github.com/dalibo/ldap2pg/internal/normalize"
	"github.com/dalibo/ldap2pg/internal/pyfmt"
	"golang.org/x/exp/maps"
)

// NormalizeGrantRule from loose YAML
//
// Sets default values. Checks some conflicts.
// Hormonize types for DuplicateGrantRules.
func NormalizeGrantRule(yaml interface{}) (rule map[string]interface{}, err error) {
	rule = map[string]interface{}{
		"owners":    "__auto__",
		"schemas":   "__all__",
		"databases": "__all__",
	}

	yamlMap, ok := yaml.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type")
	}

	err = normalize.Alias(yamlMap, "owners", "owner")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "privileges", "privilege")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "databases", "database")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "schemas", "schema")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "to")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "grantee")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "roles", "role")
	if err != nil {
		return
	}
	err = normalize.Alias(yamlMap, "objects", "object")
	if err != nil {
		return
	}

	maps.Copy(rule, yamlMap)

	keys := []string{"owners", "privileges", "databases", "schemas", "roles", "objects"}
	for _, k := range keys {
		rule[k], err = normalize.StringList(rule[k])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
	}
	err = normalize.SpuriousKeys(rule, keys...)
	return
}

// DuplicateGrantRules split plurals for mapstructure
func DuplicateGrantRules(yaml map[string]interface{}) (rules []interface{}) {
	keys := []string{"owners", "databases", "schemas", "roles", "objects", "privileges"}
	keys = lists.Filter(keys, func(s string) bool {
		return len(yaml[s].([]string)) > 0
	})
	fields := [][]string{}
	for _, k := range keys {
		fields = append(fields, yaml[k].([]string))
	}
	for combination := range lists.Product(fields...) {
		rule := map[string]interface{}{}
		for i, k := range keys {
			rule[strings.TrimSuffix(k, "s")] = combination[i]
		}
		rules = append(rules, rule)
	}
	return
}

// GrantRule is a template to generate wanted GRANTS from data
//
// data comes from LDAP search result or static configuration.
type GrantRule struct {
	Owner     pyfmt.Format
	Privilege pyfmt.Format
	Database  pyfmt.Format
	Schema    pyfmt.Format
	Object    pyfmt.Format
	To        pyfmt.Format `mapstructure:"role"`
}

func (r GrantRule) IsStatic() bool {
	return lists.And(r.Formats(), func(f pyfmt.Format) bool { return f.IsStatic() })
}

func (r GrantRule) Formats() []pyfmt.Format {
	return []pyfmt.Format{r.Owner, r.Privilege, r.Database, r.Schema, r.Object, r.To}
}

func (r GrantRule) Generate(results *ldap.Result) <-chan Grant {
	ch := make(chan Grant)
	go func() {
		defer close(ch)
		if nil == results.Entry {
			profile := r.Privilege.Input
			for _, priv := range profiles[profile] {
				// Case static rule.
				grant := Grant{
					Target:   priv.On,
					Grantee:  r.To.Input,
					Type:     priv.Type,
					Database: r.Database.Input,
					Schema:   r.Schema.Input,
					Object:   r.Object.Input,
				}
				if priv.IsDefault() {
					grant.Owner = r.Owner.Input
					grant.Object = ""
					if "global" == priv.Default {
						grant.Schema = ""
					} else if "__all__" == grant.Schema {
						// Use global default instead
						continue
					}
				}
				ch <- grant
			}
		} else {
			// Case dynamic rule.
			for values := range results.GenerateValues(r.Privilege, r.Database, r.Schema, r.Object, r.To) {
				profile := r.Privilege.Format(values)
				for _, priv := range profiles[profile] {
					grant := Grant{
						Target:   priv.On,
						Grantee:  r.To.Format(values),
						Type:     priv.Type,
						Database: r.Database.Format(values),
						Schema:   r.Schema.Format(values),
						Object:   r.Object.Format(values),
					}
					if priv.IsDefault() {
						grant.Owner = r.Owner.Input
						grant.Object = ""
						if "global" == priv.Default {
							grant.Schema = ""
						} else if "__all__" == grant.Schema {
							// Use global default instead
							continue
						}
					}
					ch <- grant
				}
			}
		}
	}()
	return ch
}
