package role

import (
	"log/slog"

	mapset "github.com/deckarep/golang-set/v2"
)

type Map map[string]Role

func (m Map) Check() error {
	for _, role := range m {
		err := role.Check(m, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m Map) Flatten() []string {
	var names []string
	seen := mapset.NewSet[string]()
	for _, role := range m {
		names = append(names, m.flattenRole(role, &seen)...)
	}
	return names
}

func (m Map) flattenRole(r Role, seen *mapset.Set[string]) []string {
	var names []string
	if (*seen).Contains(r.Name) {
		return names
	}
	for _, membership := range r.Parents {
		parent, ok := m[membership.Name]
		if !ok {
			slog.Debug("Role inherits unmanaged parent.", "role", r.Name, "parent", membership.Name)
			continue
		}
		names = append(names, m.flattenRole(parent, seen)...)

		(*seen).Add(r.Name)
	}
	names = append(names, r.Name)
	return names
}
