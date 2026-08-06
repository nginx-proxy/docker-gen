package template

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name      string
	input     string
	wantValue any
}

func TestNginxQuote(t *testing.T) {
	testCases := []testCase{}

	for _, char := range []string{"{", "}", ";", "#", " ", "'"} {
		testCases = append(
			testCases,
			testCase{
				name:      fmt.Sprintf("string with unescaped %q is quoted", char),
				input:     fmt.Sprintf(`foo%sbar`, char),
				wantValue: fmt.Sprintf(`"foo%sbar"`, char),
			},
			testCase{
				name:      fmt.Sprintf("string with incorrectly escaped %q is quoted", char),
				input:     fmt.Sprintf(`foo\\%sbar`, char),
				wantValue: fmt.Sprintf(`"foo\\%sbar"`, char),
			},
			testCase{
				name:      fmt.Sprintf("quoted string with unescaped %q is untouched", char),
				input:     fmt.Sprintf(`"foo%sbar"`, char),
				wantValue: fmt.Sprintf(`"foo%sbar"`, char),
			},
			testCase{
				name:      fmt.Sprintf("string with escaped %q is untouched", char),
				input:     fmt.Sprintf(`foo\%sbar`, char),
				wantValue: fmt.Sprintf(`foo\%sbar`, char),
			},
		)
	}

	testCases = append(
		testCases,
		testCase{
			name:      `string with unescaped " is quoted`,
			input:     `foo"bar`,
			wantValue: `'foo"bar'`,
		},
		testCase{
			name:      `string with incorrectly escaped " is quoted`,
			input:     `foo\\"bar`,
			wantValue: `'foo\\"bar'`,
		},
		testCase{
			name:      `quoted string with unescaped " is untouched`,
			input:     `'foo"bar'`,
			wantValue: `'foo"bar'`,
		},
		testCase{
			name:      `string with escaped " is untouched`,
			input:     `foo\"bar`,
			wantValue: `foo\"bar`,
		},
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := nginxQuote(tc.input)
			assert.NoError(t, err, "nginxQuote(%q) returned an unexpected error: %v", tc.input, err)
			assert.Equal(t, tc.wantValue, got, "nginxQuote(%q) returned %q; want %q", tc.input, got, tc.wantValue)
		})
	}

	t.Run("empty string returns empty string", func(t *testing.T) {
		got, err := nginxQuote("")
		assert.NoError(t, err, "nginxQuote(\"\") returned an unexpected error: %v", err)
		assert.Empty(t, got, "nginxQuote(\"\") returned %q; want empty string", got)
	})

	t.Run("string with line breaks returns error", func(t *testing.T) {
		got, err := nginxQuote("foo\nbar")
		assert.Error(t, err, "nginxQuote(\"foo\\nbar\") did not return an error for a string with line breaks")
		assert.Empty(t, got, "nginxQuote(\"foo\\nbar\") returned %q; want empty string", got)
	})

	t.Run("string with both unescaped single and double quotes returns error", func(t *testing.T) {
		got, err := nginxQuote(`foo'bar"baz`)
		assert.Error(t, err, "nginxQuote(`foo'bar\"baz`) did not return an error for a string with both unescaped single and double quotes")
		assert.Empty(t, got, "nginxQuote(`foo'bar\"baz`) returned %q; want empty string", got)
	})
}

func TestHasUnescapedNginxSpecialChars(t *testing.T) {
	testCases := []testCase{}

	for _, char := range []string{"{", "}", ";", "#", " "} {
		testCases = append(
			testCases,
			testCase{
				name:      fmt.Sprintf("unescaped %q is detected", char),
				input:     fmt.Sprintf(`foo%sbar`, char),
				wantValue: true,
			},
			testCase{
				name:      fmt.Sprintf("unescaped %q at the beginning is detected", char),
				input:     fmt.Sprintf(`%sfoobar`, char),
				wantValue: true,
			},
			testCase{
				name:      fmt.Sprintf("escaped %q is not detected", char),
				input:     fmt.Sprintf(`foo\%sbar`, char),
				wantValue: false,
			},
			testCase{
				name:      fmt.Sprintf("unescaped %q in quoted string is not detected", char),
				input:     fmt.Sprintf(`"foo%sbar"`, char),
				wantValue: false,
			},
		)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasUnescapedNginxSpecialChars(tc.input)
			assert.Equal(t, tc.wantValue, got, "hasUnescapedNginxSpecialChars(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestHasUnescapedSingleQuote(t *testing.T) {
	testCases := []testCase{
		{
			name:      "single unescaped single quote is detected",
			input:     `'`,
			wantValue: true,
		},
		{
			name:      "unescaped single quote in the middle is detected",
			input:     `foo'bar`,
			wantValue: true,
		},
		{
			name:      "unescaped single quote at the beginning is detected",
			input:     `'foobar`,
			wantValue: true,
		},
		{
			name:      "unescaped single quote at the end is detected",
			input:     `foobar'`,
			wantValue: true,
		},
		{
			name:      "unescaped single quote at the beginning with escaped ending is detected",
			input:     `'foobar\'`,
			wantValue: true,
		},
		{
			name:      "unescaped single quote in the middle of a single quoted string is detected",
			input:     `'foo'bar'`,
			wantValue: true,
		},
		{
			name:      "escaped single quote is not detected",
			input:     `foo\'bar`,
			wantValue: false,
		},
		{
			name:      "escaped single quote at the end is not detected",
			input:     `foobar\'`,
			wantValue: false,
		},
		{
			name:      "unescaped single quote in the middle of a double quoted string is not detected",
			input:     `"foo'bar"`,
			wantValue: false,
		},
		{
			name:      "empty string is not detected",
			input:     ``,
			wantValue: false,
		},
		{
			name:      "single quoted empty string is not detected",
			input:     `''`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasUnescapedQuotingChar(tc.input, '\'')
			assert.Equal(t, tc.wantValue, got, "hasUnescapedSingleQuote(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestHasUnescapedDoubleQuote(t *testing.T) {
	testCases := []testCase{
		{
			name:      "single unescaped double quote is detected",
			input:     `"`,
			wantValue: true,
		},
		{
			name:      "unescaped double quote in the middle is detected",
			input:     `foo"bar`,
			wantValue: true,
		},
		{
			name:      "unescaped double quote at the beginning is detected",
			input:     `"foobar`,
			wantValue: true,
		},
		{
			name:      "unescaped double quote at the end is detected",
			input:     `foobar"`,
			wantValue: true,
		},
		{
			name:      "unescaped double quote at the beginning with escaped ending is detected",
			input:     `"foobar\"`,
			wantValue: true,
		},
		{
			name:      "unescaped double quote inside a double quoted string is detected",
			input:     `"foo"bar"`,
			wantValue: true,
		},
		{
			name:      "escaped double quote is not detected",
			input:     `foo\"bar`,
			wantValue: false,
		},
		{
			name:      "escaped double quote at the beginning is not detected",
			input:     `\"foobar`,
			wantValue: false,
		},
		{
			name:      "escaped double quote at the end is not detected",
			input:     `foobar\"`,
			wantValue: false,
		},
		{
			name:      "unescaped double quote inside a single quoted string is not detected",
			input:     `'foo"bar'`,
			wantValue: false,
		},
		{
			name:      "empty string is not detected",
			input:     ``,
			wantValue: false,
		},
		{
			name:      "double quoted empty string is not detected",
			input:     `""`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasUnescapedQuotingChar(tc.input, '"')
			assert.Equal(t, tc.wantValue, got, "hasUnescapedDoubleQuote(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestIsSingleQuoted(t *testing.T) {
	testCases := []testCase{
		{
			name:      "single quoted string is recognized",
			input:     `'quoted'`,
			wantValue: true,
		},
		{
			name:      "empty single quoted string is recognized",
			input:     `''`,
			wantValue: true,
		},
		{
			name:      "single character string is not recognized",
			input:     `'`,
			wantValue: false,
		},
		{
			name:      "double quoted string is not recognized",
			input:     `"quoted"`,
			wantValue: false,
		},
		{
			name:      "unquoted string is not recognized",
			input:     `unquoted`,
			wantValue: false,
		},
		{
			name:      "string quoted with an escaped single quote is not recognized",
			input:     `'unquoted\'`,
			wantValue: false,
		},
		{
			name:      "incorrectly quoted string is not recognized",
			input:     `'unquoted"`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isQuotedWith(tc.input, '\'')
			assert.Equal(t, tc.wantValue, got, "isSingleQuoted(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestIsDoubleQuoted(t *testing.T) {
	testCases := []testCase{
		{
			name:      "double quoted string is recognized",
			input:     `"quoted"`,
			wantValue: true,
		},
		{
			name:      "empty double quoted string is recognized",
			input:     `""`,
			wantValue: true,
		},
		{
			name:      "single character string is not recognized",
			input:     `"`,
			wantValue: false,
		},
		{
			name:      "single quoted string is not recognized",
			input:     `'quoted'`,
			wantValue: false,
		},
		{
			name:      "unquoted string is not recognized",
			input:     `unquoted`,
			wantValue: false,
		},
		{
			name:      "string quoted with an escaped double quote is not recognized",
			input:     `"unquoted\"`,
			wantValue: false,
		},
		{
			name:      "incorrectly quoted string is not recognized",
			input:     `"unquoted'`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isQuotedWith(tc.input, '"')
			assert.Equal(t, tc.wantValue, got, "isDoubleQuoted(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestIsQuoted(t *testing.T) {
	testCases := []testCase{
		{
			name:      "double quoted string is recognized",
			input:     `"quoted"`,
			wantValue: true,
		},
		{
			name:      "single quoted string is recognized",
			input:     `'quoted'`,
			wantValue: true,
		},
		{
			name:      "unquoted string is not recognized",
			input:     `unquoted`,
			wantValue: false,
		},
		{
			name:      "incorrectly quoted string is not recognized",
			input:     `"unquoted'`,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isQuoted(tc.input)
			assert.Equal(t, tc.wantValue, got, "isQuoted(%q) returned %v; want %v", tc.input, got, tc.wantValue)
		})
	}
}

func TestIsEscapedAt(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		index     int
		wantValue bool
	}{
		{
			name:      "index zero is never escaped",
			input:     `\a`,
			index:     0,
			wantValue: false,
		},
		{
			name:      "negative index is never escaped",
			input:     `\a`,
			index:     -1,
			wantValue: false,
		},
		{
			name:      "index beyond length is never escaped",
			input:     `abc`,
			index:     4,
			wantValue: false,
		},
		{
			name:      "character with no preceding backslash is not escaped",
			input:     `abc`,
			index:     2,
			wantValue: false,
		},
		{
			name:      "character preceded by one backslash is escaped",
			input:     `a\b`,
			index:     2,
			wantValue: true,
		},
		{
			name:      "character preceded by two backslashes is not escaped",
			input:     `a\\b`,
			index:     3,
			wantValue: false,
		},
		{
			name:      "character preceded by three backslashes is escaped",
			input:     `a\\\b`,
			index:     4,
			wantValue: true,
		},
		{
			name:      "character at end of string preceded by one backslash is escaped",
			input:     `foo\"`,
			index:     4,
			wantValue: true,
		},
		{
			name:      "character at end of string preceded by two backslashes is not escaped",
			input:     `foo\\"`,
			index:     5,
			wantValue: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := isEscapedAt(tc.input, tc.index)
			assert.Equal(t, tc.wantValue, got, "isEscapedAt(%q, %d) returned %v; want %v", tc.input, tc.index, got, tc.wantValue)
		})
	}
}
