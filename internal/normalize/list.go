package normalize

import (
	"errors"
	"fmt"
)

// List ensure yaml is a list.
//
// Wraps scalar or map in a list. Returns list as is.
func List(yaml interface{}) (list []interface{}) {
	switch v := yaml.(type) {
	case []interface{}:
		list = v
	case []string:
		for _, s := range v {
			list = append(list, s)
		}
	default:
		list = append(list, yaml)
	}
	return
}

// StringList ensure yaml is a list of string.
//
// Like List, but enforce string type for items.
func StringList(yaml interface{}) (list []string, err error) {
	switch yaml.(type) {
	case nil:
		return
	case string:
		list = append(list, yaml.(string))
	case []interface{}:
		for _, iItem := range yaml.([]interface{}) {
			item, ok := iItem.(string)
			if !ok {
				return nil, errors.New("must be string")
			}
			list = append(list, item)
		}
	case []string:
		list = yaml.([]string)
	default:
		return nil, fmt.Errorf("must be string or list of string, got %v", yaml)
	}
	return
}
