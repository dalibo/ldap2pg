package normalize

// Boolean sanitizes for mapstructure.
//
// Returns "true" or "false" for common boolean values.
// Unknown values are returned as is for mapstructure validation.
func Boolean(v any) any {
	switch v {
	case "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
		return "true"
	case "n", "N", "no", "No", "NO", "off", "Off", "OFF":
		return "false"
	default:
		return v
	}
}
