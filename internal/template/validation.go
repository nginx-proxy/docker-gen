package template

import (
	"fmt"
	"slices"
)

// mustBeOneOf validates that value matches one of the allowed string values.
func mustBeOneOf(allowed []any, value string) (string, error) {
	strAllowed := make([]string, len(allowed))

	for i, candidate := range allowed {
		strAllowedValue, ok := candidate.(string)
		if !ok {
			return "", fmt.Errorf("allowed value %v (type %T) is not a string", candidate, candidate)
		}
		strAllowed[i] = strAllowedValue
	}

	if slices.Contains(strAllowed, value) {
		return value, nil
	}

	return "", fmt.Errorf(
		"value must be one of %q; got %q",
		strAllowed,
		value,
	)
}
