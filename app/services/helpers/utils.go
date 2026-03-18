package helpers

// PtrStr returns a pointer to the given string.
// Useful for initializing structs with *string fields.
func PtrStr(s string) *string {
	return &s
}
