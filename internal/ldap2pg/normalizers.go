package ldap2pg

import (
	"errors"
)

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
