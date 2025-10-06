package builder

import "encoding/json"

// deepCopy performs a deep copy of any type using JSON marshaling/unmarshaling
func deepCopy[T any](src T) T {
	// Marshal to JSON
	data, err := json.Marshal(src)
	if err != nil {
		// If marshaling fails, return a zero value
		var zero T
		return zero
	}

	// Unmarshal to a new instance
	var dst T
	if err := json.Unmarshal(data, &dst); err != nil {
		// If unmarshaling fails, return a zero value
		var zero T
		return zero
	}

	return dst
}
