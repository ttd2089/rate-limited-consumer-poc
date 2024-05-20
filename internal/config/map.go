package config

// Map is a map containing configuration values.
type Map interface {

	// Lookup looks up a single value with a complete key.
	Lookup(key string) (string, bool)
}

// A StdMap is a map[string]string.
type StdMap map[string]string

func (m StdMap) Lookup(key string) (string, bool) {
	found, ok := m[key]
	return found, ok
}
