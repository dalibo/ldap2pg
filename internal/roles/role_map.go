package roles

import (
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/slog"
)

type RoleMap map[string]Role

func (rs RoleMap) Flatten() []string {
	var names []string
	seen := mapset.NewSet[string]()
	for _, role := range rs {
		for name := range rs.flattenRole(role, &seen) {
			names = append(names, name)
		}
	}
	return names
}

func (rs RoleMap) flattenRole(r Role, seen *mapset.Set[string]) (ch chan string) {
	ch = make(chan string)
	go func() {
		defer close(ch)
		if (*seen).Contains(r.Name) {
			return
		}
		for parentName := range r.Parents.Iter() {
			parent, ok := rs[parentName]
			if !ok {
				slog.Debug("Role herits from unknown parent.", "role", r.Name, "parent", parentName)
				continue
			}
			for deepName := range rs.flattenRole(parent, seen) {
				ch <- deepName
			}
		}

		(*seen).Add(r.Name)
		ch <- r.Name
	}()
	return
}
