// fnmatch pattern list
package lists

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

func (bl *Blacklist) MatchString(item string) string {
	for _, pattern := range *bl {
		if fnmatch.Match(pattern, item, 0) {
			return pattern
		}
	}
	return ""
}

func (bl *Blacklist) Match(item Blacklistable) string {
	return bl.MatchString(item.BlacklistKey())
}
