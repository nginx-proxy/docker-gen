package template

import (
	"fmt"
	"slices"
	"strings"
)

// nginxQuote trim then quotes a string if it contains unescaped Nginx special characters or unescaped quotes.
// It returns an error if the string contains line breaks or both unescaped single and double quotes.
func nginxQuote(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", nil
	}

	// error if the value contains line breaks
	lineBreaks := []string{"\n", "\r"}
	for _, char := range lineBreaks {
		if strings.Contains(value, char) {
			return "", fmt.Errorf("value must not contain line breaks: %s", value)
		}
	}

	// error if the value contains both unescaped single and double quotes
	hasUnescapedSingleQuote := hasUnescapedSingleQuote(value)
	hasUnescapedDoubleQuote := hasUnescapedDoubleQuote(value)
	if hasUnescapedSingleQuote && hasUnescapedDoubleQuote {
		return "", fmt.Errorf("value has both unescaped single and double quotes: %s", value)
	}

	shouldBeQuoted := hasUnescapedNginxSpecialChars(value) || hasUnescapedSingleQuote || hasUnescapedDoubleQuote

	if shouldBeQuoted {
		// quote the value with single quotes if it contains unescaped double quotes, otherwise use double quotes
		if hasUnescapedDoubleQuote {
			return fmt.Sprintf("'%s'", value), nil
		}
		return fmt.Sprintf("\"%s\"", value), nil
	}

	return value, nil
}

// hasUnescapedNginxSpecialChars checks if the given string contains
// any unescaped Nginx special characters and is not enclosed in quotes.
func hasUnescapedNginxSpecialChars(value string) bool {
	nginxSpecialChars := []rune{'{', '}', ';', '#', ' '}
	for i, char := range value {
		if slices.Contains(nginxSpecialChars, char) {
			if i == 0 {
				// leading nginx special character found
				return true
			} else if !isEscapedAt(value, i) {
				// unescaped nginx special character found
				return !isQuoted(value)
			}
		}
	}
	return false
}

// hasUnescapedSingleQuote checks if the given string contains an unescaped single quote character.
func hasUnescapedSingleQuote(value string) bool {
	length := len(value)

	for i, char := range value {
		if char == '\'' {
			switch i {
			case 0: // leading single quote
				if length < 2 {
					// value is a single quote character
					return true
				}
				if value[length-1] != '\'' || isEscapedAt(value, length-1) {
					// is found without a matching unescaped trailing single quote
					return true
				}
			case length - 1: // trailing single quote
				if !isEscapedAt(value, i) && value[0] != '\'' {
					// is not escaped and found without a matching leading single quote
					return true
				}
			default:
				if !isEscapedAt(value, i) {
					// unescaped single quote is found in the middle of the string
					return !isDoubleQuoted(value)
				}
			}
		}
	}

	return false
}

// hasUnescapedDoubleQuote checks if the given string contains an unescaped double quote character.
func hasUnescapedDoubleQuote(value string) bool {
	length := len(value)

	for i, char := range value {
		if char == '"' {
			switch i {
			case 0: // leading double quote
				if length < 2 {
					// value is a double quote character
					return true
				}
				if value[length-1] != '"' || isEscapedAt(value, length-1) {
					// is found without a matching unescaped trailing double quote
					return true
				}
			case length - 1: // trailing double quote
				if !isEscapedAt(value, i) && value[0] != '"' {
					// is not escaped and found without a matching leading double quote
					return true
				}
			default:
				if !isEscapedAt(value, i) {
					// unescaped double quote is found in the middle of the string
					return !isSingleQuoted(value)
				}
			}
		}
	}

	return false
}

// isQuoted checks if the given string is enclosed in single or double quotes and the trailing quote is not escaped.
func isQuoted(value string) bool {
	return isSingleQuoted(value) || isDoubleQuoted(value)
}

// isSingleQuoted checks if the given string is enclosed in single quotes and the trailing quote is not escaped.
func isSingleQuoted(value string) bool {
	return isQuotedWith(value, '\'')
}

// isDoubleQuoted checks if the given string is enclosed in double quotes and the trailing quote is not escaped.
func isDoubleQuoted(value string) bool {
	return isQuotedWith(value, '"')
}

// isQuotedWith checks if the given string is enclosed in the specified quote character and the trailing quote is not escaped.
func isQuotedWith(value string, quoteChar rune) bool {
	length := len(value)

	if length < 2 {
		return false
	}

	hasLeadingQuote := strings.HasPrefix(value, string(quoteChar))
	hasTrailingUnescapedQuote := strings.HasSuffix(value, string(quoteChar)) && !isEscapedAt(value, length-1)
	return hasLeadingQuote && hasTrailingUnescapedQuote
}

// isEscapedAt checks whether the character at index i is escaped by an odd-length run of preceding backslashes.
func isEscapedAt(value string, i int) bool {
	if i <= 0 || i > len(value) {
		return false
	}

	precedingBackslashes := 0
	for j := i - 1; j >= 0 && value[j] == '\\'; j-- {
		precedingBackslashes++
	}

	return precedingBackslashes%2 == 1
}
