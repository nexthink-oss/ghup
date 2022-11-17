package util

// Coalesce returns the first not empty string
func Coalesce(values ...string) (value string) {
	for _, value = range values {
		if value != "" {
			return
		}
	}
	return
}
