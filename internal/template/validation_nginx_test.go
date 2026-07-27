package template

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parserExpectation struct {
	wantValue  string
	wantErr    bool
	errSnippet string
}

func assertParserResult(t *testing.T, call string, got string, err error, expect parserExpectation) {
	t.Helper()

	if expect.wantErr {
		assert.ErrorContains(t, err, expect.errSnippet, "%q: expected error containing %q, got: %v", call, expect.errSnippet, err)
		assert.Empty(t, got, "%q: expected empty result when error occurs, got: %q", call, got)
		return
	}

	assert.NoError(t, err, "%q: unexpected error: %v", call, err)
	assert.Equal(t, expect.wantValue, got, "%q: unexpected result, got %q, want %q", call, got, expect.wantValue)
}

func TestMustParseHSTS(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "off value is accepted",
			input:     "off",
			wantValue: "off",
		},
		{
			name:      "max-age only value is accepted",
			input:     "max-age=300",
			wantValue: "max-age=300",
		},
		{
			name:      "includeSubDomains is accepted",
			input:     "max-age=300; includeSubDomains",
			wantValue: "max-age=300; includeSubDomains",
		},
		{
			name:      "preload value is accepted",
			input:     "max-age=31536000; includeSubDomains; preload",
			wantValue: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name:      "whitespace is normalized",
			input:     " max-age=31536000 ; includeSubDomains ; preload ",
			wantValue: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name:       "empty value is rejected",
			input:      "",
			wantErr:    true,
			errSnippet: "cannot be empty",
		},
		{
			name:       "unknown directive is rejected",
			input:      "max-age=300; server {}",
			wantErr:    true,
			errSnippet: "unknown HSTS directive: server {}",
		},
		{
			name:       "invalid max-age format is rejected",
			input:      "max-age=abc",
			wantErr:    true,
			errSnippet: "invalid max-age directive",
		},
		{
			name:       "missing max-age is rejected",
			input:      "includeSubDomains",
			wantErr:    true,
			errSnippet: "max-age directive must be greater than 0",
		},
		{
			name:       "non-positive max-age is rejected",
			input:      "max-age=0",
			wantErr:    true,
			errSnippet: "must be greater than 0",
		},
		{
			name:       "preload without includeSubDomains is rejected",
			input:      "max-age=31536000; preload",
			wantErr:    true,
			errSnippet: "requires includeSubDomains",
		},
		{
			name:       "preload with short max-age is rejected",
			input:      "max-age=31535999; includeSubDomains; preload",
			wantErr:    true,
			errSnippet: "at least 31536000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustParseHSTS(tc.input)
			call := fmt.Sprintf("mustParseHSTS(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}
