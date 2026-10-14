package tree

import (
	"fmt"
	"maps"
	"slices"

	mapset "github.com/deckarep/golang-set/v2"
)

// Walk returns the list of string in topological order.
//
// heritance maps entity -> list of parents.
// Returns an error if heritance contains a cycle.
func Walk(heritance map[string][]string) ([]string, error) {
	done := mapset.NewSet[string]()
	visiting := mapset.NewSet[string]()
	keys := slices.Sorted(maps.Keys(heritance))
	var out []string
	for _, key := range keys {
		part, err := walkOne(key, heritance, visiting, done)
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	return out, nil
}

func walkOne(name string, groups map[string][]string, visiting, done mapset.Set[string]) ([]string, error) {
	if done.Contains(name) {
		return nil, nil
	}
	if visiting.Contains(name) {
		return nil, fmt.Errorf("loop involving %s", name)
	}

	visiting.Add(name)

	parents := slices.Clone(groups[name])
	slices.Sort(parents)

	var order []string
	for _, parent := range parents {
		part, err := walkOne(parent, groups, visiting, done)
		if err != nil {
			return nil, err
		}
		order = append(order, part...)
	}

	visiting.Remove(name)
	done.Add(name)
	order = append(order, name)
	return order, nil
}
