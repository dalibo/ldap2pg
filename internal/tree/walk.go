package tree

import (
	"slices"

	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/exp/maps"
)

// Walk returns the list of string in topological order.
//
// heritance maps entity -> list of parents.
func Walk(heritance map[string][]string) (out []string) {
	seen := mapset.NewSet[string]()
	keys := maps.Keys(heritance)
	slices.Sort(keys)
	for _, key := range keys {
		out = append(out, walkOne(key, heritance, &seen)...)
	}
	return
}

func walkOne(name string, groups map[string][]string, seen *mapset.Set[string]) (order []string) {
	if (*seen).Contains(name) {
		return nil
	}

	parents := groups[name]
	slices.Sort(parents)

	for _, parent := range groups[name] {
		order = append(order, walkOne(parent, groups, seen)...)
	}
	order = append(order, name)
	(*seen).Add(name)
	return
}
