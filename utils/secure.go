package utils

import "strconv"

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

// StrToInt converts a string to an integer, returning 0 if the conversion fails.
func StrToInt(s string) int {
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return result
}

// StrToUint converts a string to an unsigned integer, returning 0 if the conversion fails.
func StrToUint(s string) uint {
	result := StrToInt(s)
	if result < 0 {
		return 0
	}
	return uint(result)
}

// BoolToUint converts a boolean to an unsigned integer, returning 1 for true and 0 for false.
func BoolToUint(b bool) uint {
	if b {
		return 1
	}
	return 0
}
