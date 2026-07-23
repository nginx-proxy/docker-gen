package template

import "testing"

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
