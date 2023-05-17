package roles

import (
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slog"
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
		for parentName := range r.Parents.Iter() {
			parent, ok := m[parentName]
			if !ok {
				slog.Debug("Role herits unmanaged parent.", "role", r.Name, "parent", parentName)
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
