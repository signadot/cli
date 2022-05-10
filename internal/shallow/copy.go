package shallow

// Copy returns a pointer to a new instance of the given type, initialized with the given value.
func Copy[T any](v T) *T {
	return &v
}
