// fnmatch pattern list
package ldap2pg

import (
	"github.com/tzvetkoff-go/fnmatch"
)

type (
	Blacklist     []string
	Blacklistable interface {
		BlacklistKey() string
	}
)

func (bl *Blacklist) Filter(items []Blacklistable) []Blacklistable {
	var filteredItems []Blacklistable
	for _, item := range items {
		match := bl.Match(item)
		if match == "" {
			filteredItems = append(filteredItems, item)
		}
	}
	return filteredItems
}

func (bl *Blacklist) Match(item Blacklistable) string {
	key := item.BlacklistKey()
	for _, pattern := range *bl {
		if fnmatch.Match(pattern, key, 0) {
			return pattern
		}
	}
	return ""
}
