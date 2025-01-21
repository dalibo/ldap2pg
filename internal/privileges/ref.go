package privileges

import (
	"log/slog"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slices"
)

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

func (rm RefMap) BuildTypeMaps() (instance, other, defaults TypeMap) {
	all := make(TypeMap)
	other = make(TypeMap)
	defaults = make(TypeMap)
	instance = make(TypeMap)

	for _, privList := range rm {
		for _, priv := range privList {
			var k, t string
			if "" != priv.Default {
				k = strings.ToUpper(priv.Default) + " DEFAULT"
				t = priv.On + "--" + priv.Type
			} else {
				k = priv.On
				t = priv.Type
			}

			all[k] = append(all[k], t)
		}
	}

	for target, types := range all {
		set := mapset.NewSet(types...)
		types := set.ToSlice()
		slices.Sort(types)
		if strings.HasSuffix(target, " DEFAULT") {
			defaults[target] = types
		} else if Builtins[target].IsGlobal() {
			instance[target] = types
		} else {
			other[target] = types
		}
		slog.Debug("Managing privileges.", "types", types, "on", target)
	}

	return
}
