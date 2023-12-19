package utils

import (
	"os"
	"strings"
)

// SplitKeyValueSlice takes a string slice where values are of the form
// KEY, KEY=, KEY=VALUE  or KEY=NESTED_KEY=VALUE2, and returns a map[string]string where items
// are split at their first `=`.
func SplitKeyValueSlice(in []string) map[string]string {
	env := make(map[string]string)
	for _, entry := range in {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			parts = append(parts, "")
		}
		env[parts[0]] = parts[1]
	}
	return env

}

// PathExists returns whether the given file or directory exists or not
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
