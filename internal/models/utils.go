package models

// cloneModelSet creates a shallow copy of a model set map.
// This is used to safely return cached model sets without exposing
// the internal cache to concurrent modifications.
func cloneModelSet(input map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(input))
	for key := range input {
		out[key] = struct{}{}
	}
	return out
}
