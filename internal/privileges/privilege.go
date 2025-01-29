package privileges

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/dalibo/ldap2pg/internal/normalize"
)

// Privilege references a privilege type and an ACL
//
// Example: {Type: "CONNECT", To: "DATABASE"}
type Privilege struct {
	Type   string // Privilege type (USAGE, etc.)
	On     string // ACL (DATABASE, GLOBAL DEFAULT, etc)
	Object string // TABLES, SCHEMAS, etc.
}

func (p Privilege) ACL() string {
	return p.On
}

func NormalizePrivilege(rawPrivilege interface{}) (interface{}, error) {
	m, ok := rawPrivilege.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("bad type")
	}

	err := normalize.Alias(m, "types", "type")
	if err != nil {
		return m, err
	}

	m["types"] = normalize.List(m["types"])

	// DEPRECATED: v6.2 compat
	def, _ := m["default"].(string)
	if def != "" {
		m["object"] = "TABLES"
		m["on"] = fmt.Sprintf("%s DEFAULT", strings.ToUpper(def))
		slog.Warn("Deprecated default scope.")
		slog.Warn("Use 'object' instead of 'default' in privilege definition.", "on", m["on"], "object", m["object"])
	}

	return m, nil
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
