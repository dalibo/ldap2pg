package privilege

// Ref references a privilege type
//
// Example: {Type: "CONNECT", To: "DATABASE"}
type Ref struct {
	Type string
	On   string
}

// RefMap holds privilege groups
type RefMap map[string][]Ref
