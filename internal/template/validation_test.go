package template

import (
	"strings"
	"testing"
)

func TestMustBeOneOf(t *testing.T) {
	testCases := []struct {
		name          string
		allowedValues []any
		input         string
		wantValue     string
		wantErr       bool
		errSnippet    string
	}{
		{
			name:          "valid input value",
			allowedValues: []any{"a", "b", "c"},
			input:         "a",
			wantValue:     "a",
		},
		{
			name:          "invalid input value",
			allowedValues: []any{"a", "b", "c"},
			input:         "d",
			wantErr:       true,
			errSnippet:    `value must be one of ["a" "b" "c"]`,
		},
		{
			name:          "non-string allowed value",
			allowedValues: []any{"a", 1, "c"},
			input:         "a",
			wantErr:       true,
			errSnippet:    "is not a string",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustBeOneOf(tc.allowedValues, tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("mustBeOneOf(%v, %q) expected an error; got nil", tc.allowedValues, tc.input)
				}
				if got != "" {
					t.Fatalf("mustBeOneOf(%v, %q) returned unexpected value on error: %q", tc.allowedValues, tc.input, got)
				}
				if tc.errSnippet != "" && !strings.Contains(err.Error(), tc.errSnippet) {
					t.Fatalf("mustBeOneOf(%v, %q) error %q does not contain %q", tc.allowedValues, tc.input, err.Error(), tc.errSnippet)
				}
				return
			}

			if err != nil {
				t.Fatalf("mustBeOneOf(%v, %q) returned unexpected error: %v", tc.allowedValues, tc.input, err)
			}
			if got != tc.wantValue {
				t.Fatalf("mustBeOneOf(%v, %q) returned %q; want %q", tc.allowedValues, tc.input, got, tc.wantValue)
			}
		})
	}
}

func TestMustBeInt(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "valid zero value",
			input:     "0",
			wantValue: "0",
		},
		{
			name:      "valid positive value",
			input:     "161",
			wantValue: "161",
		},
		{
			name:      "negative value is rejected",
			input:     "-42",
			wantValue: "-42",
		},
		{
			name:       "empty value is rejected",
			input:      "",
			wantErr:    true,
			errSnippet: "empty value",
		},
		{
			name:       "non integer value is rejected",
			input:      "abc",
			wantErr:    true,
			errSnippet: "must be an integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustBeInt(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("mustBeInt(%q) expected an error; got nil", tc.input)
				}
				if got != "" {
					t.Fatalf("mustBeInt(%q) returned unexpected value on error: %q", tc.input, got)
				}
				if tc.errSnippet != "" && !strings.Contains(err.Error(), tc.errSnippet) {
					t.Fatalf("mustBeInt(%q) error %q does not contain %q", tc.input, err.Error(), tc.errSnippet)
				}
				return
			}

			if err != nil {
				t.Fatalf("mustBeInt(%q) returned unexpected error: %v", tc.input, err)
			}
			if got != tc.wantValue {
				t.Fatalf("mustBeInt(%q) returned %q; want %q", tc.input, got, tc.wantValue)
			}
		})
	}
}

func TestMustBeIntInRange(t *testing.T) {
	testCases := []struct {
		name       string
		min        int
		max        int
		input      string
		wantValue  string
		wantErr    bool
		errSnippet string
	}{
		{
			name:      "value in range",
			min:       -3,
			max:       3,
			input:     "2",
			wantValue: "2",
		},
		{
			name:      "value at lower boundary",
			min:       -3,
			max:       3,
			input:     "-3",
			wantValue: "-3",
		},
		{
			name:      "value at upper boundary",
			min:       -3,
			max:       3,
			input:     "3",
			wantValue: "3",
		},
		{
			name:       "invalid allowed range",
			min:        5,
			max:        -10,
			input:      "3",
			wantErr:    true,
			errSnippet: "invalid allowed range",
		},
		{
			name:       "empty value is rejected",
			min:        -3,
			max:        3,
			input:      "",
			wantErr:    true,
			errSnippet: "empty value",
		},
		{
			name:       "non integer value is rejected",
			min:        -3,
			max:        3,
			input:      "abc",
			wantErr:    true,
			errSnippet: "must be an integer",
		},
		{
			name:       "value below minimum is rejected",
			min:        -3,
			max:        3,
			input:      "-4",
			wantErr:    true,
			errSnippet: "between -3 and 3",
		},
		{
			name:       "value above maximum is rejected",
			min:        -3,
			max:        3,
			input:      "4",
			wantErr:    true,
			errSnippet: "between -3 and 3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := mustBeIntInRange(tc.min, tc.max, tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("mustBeIntInRange(%d, %d, %q) expected an error; got nil", tc.min, tc.max, tc.input)
				}
				if got != "" {
					t.Fatalf("mustBeIntInRange(%d, %d, %q) returned unexpected value on error: %q", tc.min, tc.max, tc.input, got)
				}
				if tc.errSnippet != "" && !strings.Contains(err.Error(), tc.errSnippet) {
					t.Fatalf(
						"mustBeIntInRange(%d, %d, %q) error %q does not contain %q",
						tc.min,
						tc.max,
						tc.input,
						err.Error(),
						tc.errSnippet,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("mustBeIntInRange(%d, %d, %q) returned unexpected error: %v", tc.min, tc.max, tc.input, err)
			}
			if got != tc.wantValue {
				t.Fatalf("mustBeIntInRange(%d, %d, %q) returned %q; want %q", tc.min, tc.max, tc.input, got, tc.wantValue)
			}
		})
	}
}
