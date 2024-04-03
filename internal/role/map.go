package role

import (
	"log/slog"

	mapset "github.com/deckarep/golang-set/v2"
)

type Map map[string]Role

func (m Map) Flatten() []string {
	var names []string
	seen := mapset.NewSet[string]()
	for _, role := range m {
		for name := range m.flattenRole(role, &seen) {
			names = append(names, name)
		}
	}
	return names
}

func (m Map) flattenRole(r Role, seen *mapset.Set[string]) (ch chan string) {
	ch = make(chan string)
	go func() {
		defer close(ch)
		if (*seen).Contains(r.Name) {
			return
		}
		for _, membership := range r.Parents {
			parent, ok := m[membership.Name]
			if !ok {
				slog.Debug("Role herits unmanaged parent.", "role", r.Name, "parent", membership.Name)
				continue
			}
			for deepName := range m.flattenRole(parent, seen) {
				ch <- deepName
			}
		}

		(*seen).Add(r.Name)
		ch <- r.Name
	}()
	return
}
