package normalize

import "fmt"

// Alias rename a key in a map.
//
// Returns an error if alias and key already co-exists.
func Alias(yaml map[string]interface{}, key, alias string) (err error) {
	value, hasAlias := yaml[alias]
	if !hasAlias {
		return
	}

	_, hasKey := yaml[key]
	if hasKey {
		return &conflict{
			key0: key,
			key1: alias,
		}
	}

	delete(yaml, alias)
	yaml[key] = value
	return
}

type conflict struct {
	key0 string
	key1 string
}

func (err *conflict) Error() string {
	return fmt.Sprintf("key conflict between %s and %s", err.key0, err.key1)
}
