package privileges

import (
	"fmt"
	"strings"
)

// Privilege references a privilege type and an ACL
//
// Example: {Type: "CONNECT", To: "DATABASE"}
type Privilege struct {
	Default string // "", global or schema
	Type    string // Privilege type (USAGE, etc.)
	On      string // Object class (DATABASE, TABLES, etc)
}

func (p Privilege) IsDefault() bool {
	return "" != p.Default
}

func (p Privilege) ACL() string {
	if p.IsDefault() {
		return fmt.Sprintf("%s DEFAULT", strings.ToUpper(p.Default))
	}
	return p.On
}

func DuplicatePrivilege(yaml map[string]interface{}) (privileges []interface{}) {
	for _, singleType := range yaml["types"].([]interface{}) {
		privilege := make(map[string]interface{})
		privilege["type"] = singleType
		for key, value := range yaml {
			if "types" == key {
				continue
			}
			privilege[key] = value
		}
		privileges = append(privileges, privilege)
	}
	return
}
