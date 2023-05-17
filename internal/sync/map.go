package sync

// Map holds a set of rules to generate wanted state.
type Map []Item

func (m Map) HasLDAPSearches() bool {
	for _, item := range m {
		if item.HasLDAPSearch() {
			return true
		}
	}
	return false
}

func (m Map) SplitStaticRules() (newMap Map) {
	newMap = make(Map, 0)
	for _, item := range m {
		newMap = append(newMap, item.SplitStaticItems()...)
	}
	return
}
