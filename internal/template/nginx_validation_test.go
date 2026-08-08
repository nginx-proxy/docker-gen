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

func TestNginxMustParseHSTS(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
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
			name:      "function is case-insensitive",
			input:     "Max-Age=31536000; includesubdomains; PRELOAD",
			wantValue: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name:       "empty value is rejected",
			input:      " ",
			wantErr:    true,
			errSnippet: "cannot be empty",
		},
		{
			name:       "max-age with empty value is rejected",
			input:      "max-age=",
			wantErr:    true,
			errSnippet: "invalid Strict-Transport-Security max-age directive",
		},
		{
			name:       "unknown directive is rejected",
			input:      "max-age=300; server {}",
			wantErr:    true,
			errSnippet: "unknown Strict-Transport-Security directive: server {}",
		},
		{
			name:       "max-age with trailing garbage is rejected",
			input:      "max-age=300foo",
			wantErr:    true,
			errSnippet: "invalid Strict-Transport-Security max-age directive",
		},
		{
			name:       "invalid max-age format is rejected",
			input:      "max-age=abc",
			wantErr:    true,
			errSnippet: "invalid Strict-Transport-Security max-age directive",
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
			got, err := nginxMustParseHSTS(tc.input)
			call := fmt.Sprintf("mustParseHSTS(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}

func TestMustParseLoadbalance(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "hash without parameter is accepted",
			input:     "hash",
			wantValue: "hash",
		},
		{
			name:      "hash with parameter is accepted",
			input:     "hash $request_uri",
			wantValue: "hash $request_uri",
		},
		{
			name:      "ip_hash is accepted",
			input:     "ip_hash",
			wantValue: "ip_hash",
		},
		{
			name:      "least_conn is accepted",
			input:     "least_conn",
			wantValue: "least_conn",
		},
		{
			name:      "least_time with header parameter is accepted",
			input:     "least_time header",
			wantValue: "least_time header",
		},
		{
			name:      "least_time with last_byte inflight parameter is accepted",
			input:     "least_time last_byte inflight",
			wantValue: "least_time last_byte inflight",
		},
		{
			name:      "random without parameters is accepted",
			input:     "random",
			wantValue: "random",
		},
		{
			name:      "random with two parameter is accepted",
			input:     "random two",
			wantValue: "random two",
		},
		{
			name:      "trailing semicolon is normalized",
			input:     "least_conn;",
			wantValue: "least_conn",
		},
		{
			name:       "unknown directive is rejected",
			input:      "round_robin",
			wantErr:    true,
			errSnippet: "invalid loadbalance directive",
		},
		{
			name:       "ip_hash with parameter is rejected",
			input:      "ip_hash foo",
			wantErr:    true,
			errSnippet: "does not take any parameters",
		},
		{
			name:       "least_time without parameter is rejected",
			input:      "least_time",
			wantErr:    true,
			errSnippet: "requires a parameter",
		},
		{
			name:       "least_time with invalid parameter is rejected",
			input:      "least_time invalid",
			wantErr:    true,
			errSnippet: "invalid least_time loadbalance method parameter",
		},
		{
			name:       "random with unsupported first parameter is rejected",
			input:      "random one",
			wantErr:    true,
			errSnippet: "does not take any first parameter other than 'two'",
		},
		{
			name:       "random with second parameter is rejected",
			input:      "random two least_conn",
			wantErr:    true,
			errSnippet: "does not take any first parameter other than 'two'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustParseLoadbalance(tc.input)
			call := fmt.Sprintf("mustParseLoadbalance(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}

func TestParseHashLoadBalanceMethod(t *testing.T) {
	testCases := []struct {
		name       string
		input      []string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "method without parameters is accepted",
			input:     []string{"hash"},
			wantValue: "hash",
		},
		{
			name:      "method with one parameter is accepted",
			input:     []string{"hash", " $request_uri "},
			wantValue: "hash $request_uri",
		},
		{
			name:      "first parameter is quoted if it contains unescaped special characters",
			input:     []string{"hash", "foo bar{}"},
			wantValue: "hash \"foo bar{}\"",
		},
		{
			name:      "method with consistent parameter is accepted",
			input:     []string{"hash", " $request_uri ", " consistent "},
			wantValue: "hash $request_uri consistent",
		},
		{
			name:       "method with unsupported second parameter is rejected",
			input:      []string{"hash", "$request_uri", "invalid"},
			wantErr:    true,
			errSnippet: "other than 'consistent'",
		},
		{
			name:       "too many parameters are rejected",
			input:      []string{"hash", "$request_uri", "consistent", "extra"},
			wantErr:    true,
			errSnippet: "does not take more than two parameters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseHashLoadBalanceMethod(tc.input)
			call := fmt.Sprintf("parseHashLoadBalanceMethod(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}

func TestParseUnparameterizedLoadbalanceMethod(t *testing.T) {
	testCases := []struct {
		name       string
		input      []string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "single method value is accepted",
			input:     []string{" ip_hash "},
			wantValue: "ip_hash",
		},
		{
			name:       "parameters are rejected",
			input:      []string{"least_conn", "extra"},
			wantErr:    true,
			errSnippet: "does not take any parameters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseUnparameterizedLoadbalanceMethod(tc.input)
			call := fmt.Sprintf("parseUnparameterizedLoadbalanceMethod(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}

func TestParseLeastTimeLoadbalanceMethod(t *testing.T) {
	testCases := []struct {
		name       string
		input      []string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "header parameter is accepted",
			input:     []string{"least_time", "header"},
			wantValue: "least_time header",
		},
		{
			name:      "last_byte inflight parameter is accepted",
			input:     []string{"least_time", " last_byte inflight "},
			wantValue: "least_time last_byte inflight",
		},
		{
			name:       "missing parameter is rejected",
			input:      []string{"least_time"},
			wantErr:    true,
			errSnippet: "requires a parameter",
		},
		{
			name:       "too many segments are rejected",
			input:      []string{"least_time", "last_byte", "inflight"},
			wantErr:    true,
			errSnippet: "requires a parameter",
		},
		{
			name:       "invalid parameter is rejected",
			input:      []string{"least_time", "invalid"},
			wantErr:    true,
			errSnippet: "invalid least_time loadbalance method parameter",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLeastTimeLoadbalanceMethod(tc.input)
			call := fmt.Sprintf("parseLeastTimeLoadbalanceMethod(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}

func TestParseRandomLoadbalanceMethod(t *testing.T) {
	testCases := []struct {
		name       string
		input      []string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "method without parameters is accepted",
			input:     []string{"random"},
			wantValue: "random",
		},
		{
			name:      "method with two parameter is accepted",
			input:     []string{"random", " two "},
			wantValue: "random two",
		},
		{
			name:      "method with valid second parameter is accepted",
			input:     []string{"random", "two", "least_conn"},
			wantValue: "random two least_conn",
		},
		{
			name:       "unsupported first parameter is rejected",
			input:      []string{"random", "one"},
			wantErr:    true,
			errSnippet: "other than 'two'",
		},
		{
			name:       "unsupported first parameter with valid second parameter is rejected",
			input:      []string{"random", "one", "least_conn"},
			wantErr:    true,
			errSnippet: "other than 'two'",
		},
		{
			name:       "unsupported second parameter is rejected",
			input:      []string{"random", "two", "invalid"},
			wantErr:    true,
			errSnippet: "only accepts",
		},
		{
			name:       "too many parameters are rejected",
			input:      []string{"random", "two", "least_conn", "extra"},
			wantErr:    true,
			errSnippet: "does not take more than two parameters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRandomLoadbalanceMethod(tc.input)
			call := fmt.Sprintf("parseRandomLoadbalanceMethod(%q)", tc.input)
			assertParserResult(t, call, got, err, parserExpectation{
				wantValue:  tc.wantValue,
				wantErr:    tc.wantErr,
				errSnippet: tc.errSnippet,
			})
		})
	}
}
