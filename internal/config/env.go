package config

import (
	"os"
	"strings"
)

// An EnvMap is a [Map] that reads from environment variables. Keys are mapped to environment
// variable names by replacing hyphens ('-') with underscores ('_'), replacing periods ('.') with
// two underscores ("__"), and transforming the key to UPPER-CASE.
type EnvMap struct{}

func (EnvMap) Lookup(key string) (string, bool) {
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, ".", "__")
	key = strings.ToUpper(key)
	return os.LookupEnv(key)
}
