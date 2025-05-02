package utils

// BlurMap is a filter that replace the value of the given keys in the map with asterisks (*).
func BlurMap(m map[string]interface{}, s ...string) {
	for _, key := range s {
		if _, ok := m[key]; ok {
			m[key] = "*******"
		}
	}
}

// BlurStr is a filter that replace the string with asterisks (*).
func BlurStr() string {
	return "*******"
}
