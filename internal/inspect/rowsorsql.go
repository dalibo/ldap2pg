package inspect

// Either an SQL string or a predefined list of YAML rows.
type RowsOrSQL struct {
	Value interface{}
}

// Like pgx.RowToFunc, but from YAML
type YamlToFunc[T any] func(row interface{}) (T, error)

func IsPredefined(q RowsOrSQL) bool {
	switch q.Value.(type) {
	case string:
		return false
	default:
		return true
	}
}

// Implements inspect.YamlToFunc. Similar to pgx.RowTo.
func YamlToString(value interface{}) (pattern string, err error) {
	pattern = value.(string)
	return
}
