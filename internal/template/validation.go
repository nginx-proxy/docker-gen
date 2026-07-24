package template

import (
	"fmt"
	"math"
	"slices"
	"strconv"
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

// mustBeInt validates that value is a base-10 integer string.
func mustBeInt(value string) (string, error) {
	return mustBeIntInRange(math.MinInt, math.MaxInt, value)
}

// mustBeIntInRange validates that value is a base-10 integer string within min..max (inclusive).
func mustBeIntInRange(min, max int, value string) (string, error) {
	if min > max {
		return "", fmt.Errorf("invalid allowed range %d..%d", min, max)
	}

	if value == "" {
		return "", fmt.Errorf("value must be an integer; got an empty value")
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return "", fmt.Errorf("value must be an integer; got %q", value)
	}

	if parsed < int64(min) || parsed > int64(max) {
		return "", fmt.Errorf("value must be an integer between %d and %d; got %q", min, max, value)
	}

	return value, nil
}
