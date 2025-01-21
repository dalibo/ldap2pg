package normalize

import (
	"fmt"

	"golang.org/x/exp/slices"
)

// SpuriousKeys checks for unknown keys in a YAML map.
func SpuriousKeys(yaml map[string]interface{}, knownKeys ...string) error {
	for key := range yaml {
		if !slices.Contains(knownKeys, key) {
			return fmt.Errorf("unknown key '%s'", key)
		}
	}
	return nil
}

// IsString checks for string type.
func IsString(yaml interface{}) error {
	_, ok := yaml.(string)
	if !ok && yaml != nil {
		return fmt.Errorf("bad value %v, must be string", yaml)
	}
	return nil
}
