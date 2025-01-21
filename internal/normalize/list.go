package normalize

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
