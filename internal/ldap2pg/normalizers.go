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

type ParseError struct {
	Message string
	Value   interface{}
}

func (err *ParseError) Error() string {
	return err.Message
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

func NormalizeRoleRule(yaml interface{}) (rule map[string]interface{}, err error) {
	var names []string
	switch yaml.(type) {
	case string:
		rule = make(map[string]interface{})
		names = append(names, yaml.(string))
		rule["names"] = names
	case map[string]interface{}:
		rule = yaml.(map[string]interface{})
		err = NormalizeAlias(&rule, "names", "name")
		if err != nil {
			return
		}
		names, ok := rule["names"]
		if ok {
			rule["names"], err = NormalizeStringList(names)
			if err != nil {
				return
			}
		} else {
			err = errors.New("Missing name in role rule")
			return
		}
		err = NormalizeAlias(&rule, "comments", "comment")
		if err != nil {
			return
		}
		comments, ok := rule["comments"]
		if !ok {
			comments = []interface{}{}
		}
		rule["comments"], err = NormalizeStringList(comments)
		if err != nil {
			return
		}
	default:
		err = &ParseError{
			Message: "Invalid role rule YAML",
			Value:   yaml,
		}
	}
	return
}
