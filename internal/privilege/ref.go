package privilege

// Ref references a privilege type
//
// Example: {Type: "CONNECT", To: "DATABASE"}
type Ref struct {
	Default string // "", global or schema
	Type    string // Privilege type (USAGE, etc.)
	On      string // Object class (DATABASE, TABLES, etc)
}

func (r Ref) IsDefault() bool {
	return "" != r.Default
}

// RefMap holds privilege groups
type RefMap map[string][]Ref

// BuildDefaultArg returns the list of (Type, On) couple referenced.
func (rm RefMap) BuildDefaultArg(def string) (out [][]string) {
	for _, refs := range rm {
		for _, ref := range refs {
			if ref.Default != def {
				continue
			}
			out = append(out, []string{ref.On, ref.Type})
		}
	}
	return
}
