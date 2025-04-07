package normalize

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// SpuriousKeys checks for unknown keys in a YAML map.
func SpuriousKeys(yaml map[string]any, knownKeys ...string) error {
	for key := range yaml {
		if !slices.Contains(knownKeys, key) {
			return fmt.Errorf("unknown key '%s'", key)
		}
	}
	return nil
}
