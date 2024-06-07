// fnmatch pattern list
package lists

import (
	"path/filepath"
)

type (
	Blacklist     []string
	Blacklistable interface {
		BlacklistKey() string
	}
)

// Check verify patterns are valid.
//
// Use it before using MatchString().
func (bl *Blacklist) Check() error {
	for _, pattern := range *bl {
		_, err := filepath.Match(pattern, "pouet")
		if err != nil {
			return err
		}
	}
	return nil
}

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

// MatchString returns the first pattern that matches the item.
//
// Use Check() before using MatchString().
// panics if pattern is invalid.
// returns empty string if no match.
func (bl *Blacklist) MatchString(item string) string {
	for _, pattern := range *bl {
		ok, err := filepath.Match(pattern, item)
		if err != nil {
			// Use Check() before using MatchString().
			panic(err)
		}
		if ok {
			return pattern
		}
	}
	return ""
}

func (bl *Blacklist) Match(item Blacklistable) string {
	return bl.MatchString(item.BlacklistKey())
}
