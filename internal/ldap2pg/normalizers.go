// Functions to normalize YAML input before processing into data structure.
package ldap2pg

import (
	"errors"
)

type KeyConflict struct {
	Key      string
	Conflict string
}

func (err *KeyConflict) Error() string {
	return "YAML alias conflict"
}

func NormalizeAlias(yaml *map[string]interface{}, key, alias string) (err error) {
	value, hasAlias := (*yaml)[alias]
	if !hasAlias {
		return
	}

	_, hasKey := (*yaml)[key]
	if hasKey {
		err = &KeyConflict{
			Key:      key,
			Conflict: alias,
		}
		return
	}

	delete(*yaml, alias)
	(*yaml)[key] = value
	return
}

func NormalizeList(yaml interface{}) (list []interface{}) {
	list, ok := yaml.([]interface{})
	if !ok {
		list = append(list, yaml)
	}
	return
}

func NormalizeStringList(yaml interface{}) (list []string, err error) {
	iList, ok := yaml.([]interface{})
	if !ok {
		iList = append(iList, yaml)
	}
	for _, iItem := range iList {
		item, ok := iItem.(string)
		if !ok {
			err = errors.New("Must be string")
		}
		list = append(list, item)
	}
	return
}
