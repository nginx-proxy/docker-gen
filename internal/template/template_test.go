package template

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type templateTestList []struct {
	tmpl     string
	context  interface{}
	expected interface{}
}

func (tests templateTestList) run(t *testing.T) {
	for n, test := range tests {
		test := test
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			t.Parallel()
			wantErr, _ := test.expected.(error)
			want, ok := test.expected.(string)
			if !ok && wantErr == nil {
				t.Fatalf("test bug: want a string or error for .expected, got %v", test.expected)
			}
			tmpl, err := newTemplate("testTemplate").Parse(test.tmpl)
			if err != nil {
				t.Fatalf("Template parse failed: %v", err)
			}

			var b bytes.Buffer
			err = tmpl.ExecuteTemplate(&b, "testTemplate", test.context)
			got := b.String()
			if err != nil {
				if wantErr != nil {
					return
				}
				t.Fatalf("Error executing template: %v", err)
			} else if wantErr != nil {
				t.Fatalf("Expected error, got %v", got)
			}
			if want != got {
				t.Fatalf("Incorrect output found; want %#v, got %#v", want, got)
			}
		})
	}
}

func TestGetArrayValues(t *testing.T) {
	values := []string{"foor", "bar", "baz"}
	var expectedType *reflect.Value

	arrayValues, err := getArrayValues("testFunc", values)
	assert.NoError(t, err)
	assert.IsType(t, expectedType, arrayValues)
	assert.Equal(t, "bar", arrayValues.Index(1).String())

	arrayValues, err = getArrayValues("testFunc", &values)
	assert.NoError(t, err)
	assert.IsType(t, expectedType, arrayValues)
	assert.Equal(t, "baz", arrayValues.Index(2).String())

	arrayValues, err = getArrayValues("testFunc", "foo")
	assert.Error(t, err)
	assert.Nil(t, arrayValues)
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{" ", true},
		{"   ", true},
		{"\t", true},
		{"\t\n\v\f\r\u0085\u00A0", true},
		{"a", false},
		{" a ", false},
		{"a ", false},
		{" a", false},
		{"日本語", false},
	}

	for _, i := range tests {
		v := isBlank(i.input)
		if v != i.expected {
			t.Fatalf("expected '%v'. got '%v'", i.expected, v)
		}
	}
}

func TestRemoveBlankLines(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"\r\n\r\n", ""},
		{"line1\nline2", "line1\nline2"},
		{"line1\n\nline2", "line1\nline2"},
		{"\n\n\n\nline1\n\nline2", "line1\nline2"},
		{"\n\n\n\n\n  \n \n \n", ""},

		// windows line endings \r\n
		{"line1\r\nline2", "line1\r\nline2"},
		{"line1\r\n\r\nline2", "line1\r\nline2"},

		// keep last new line
		{"line1\n", "line1\n"},
		{"line1\r\n", "line1\r\n"},
	}

	for _, i := range tests {
		output := new(bytes.Buffer)
		removeBlankLines(strings.NewReader(i.input), output)
		if output.String() != i.expected {
			t.Fatalf("expected '%v'. got '%v'", i.expected, output)
		}
	}
}

// TestSprig ensures that the migration to sprig to provide certain functions did not break
// compatibility with existing templates.
func TestSprig(t *testing.T) {
	for _, tc := range []struct {
		desc string
		tts  templateTestList
	}{
		{"dict", templateTestList{
			{`{{ $d := dict "a" "b" }}{{ if eq (index $d "a") "b" }}ok{{ end }}`, nil, `ok`},
			{`{{ $d := dict "a" "b" }}{{ if eq (index $d "x") nil }}ok{{ end }}`, nil, `ok`},
			{`{{ $d := dict "a" "b" "c" (dict "d" "e") }}{{ if eq (index $d "c" "d") "e" }}ok{{ end }}`, nil, `ok`},
		}},
		{"first", templateTestList{
			{`{{ if eq (first $) "a"}}ok{{ end }}`, []string{"a", "b"}, `ok`},
			{`{{ if eq (first $) "a"}}ok{{ end }}`, [2]string{"a", "b"}, `ok`},
		}},
		{"hasPrefix", templateTestList{
			{`{{ if hasPrefix "tcp://" "tcp://127.0.0.1:2375" }}ok{{ end }}`, nil, `ok`},
			{`{{ if not (hasPrefix "udp://" "tcp://127.0.0.1:2375") }}ok{{ end }}`, nil, `ok`},
		}},
		{"hasSuffix", templateTestList{
			{`{{ if hasSuffix ".local" "myhost.local" }}ok{{ end }}`, nil, `ok`},
			{`{{ if not (hasSuffix ".example" "myhost.local") }}ok{{ end }}`, nil, `ok`},
		}},
		{"last", templateTestList{
			{`{{ if eq (last $) "b"}}ok{{ end }}`, []string{"a", "b"}, `ok`},
			{`{{ if eq (last $) "b"}}ok{{ end }}`, [2]string{"a", "b"}, `ok`},
		}},
		{"trim", templateTestList{
			{`{{ if eq (trim "  myhost.local  ") "myhost.local" }}ok{{ end }}`, nil, `ok`},
		}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc.tts.run(t)
		})
	}
}

func TestEval(t *testing.T) {
	for _, tc := range []struct {
		desc string
		tts  templateTestList
	}{
		{"undefined", templateTestList{
			{`{{eval "missing"}}`, nil, errors.New("")},
			{`{{eval "missing" nil}}`, nil, errors.New("")},
			{`{{eval "missing" "abc"}}`, nil, errors.New("")},
			{`{{eval "missing" "abc" "def"}}`, nil, errors.New("")},
		}},
		// The purpose of the "ctx" context is to assert that $ and . inside the template is the
		// eval argument, not the global context.
		{"noArg", templateTestList{
			{`{{define "T"}}{{$}}{{.}}{{end}}{{eval "T"}}`, "ctx", "<no value><no value>"},
		}},
		{"nilArg", templateTestList{
			{`{{define "T"}}{{$}}{{.}}{{end}}{{eval "T" nil}}`, "ctx", "<no value><no value>"},
		}},
		{"oneArg", templateTestList{
			{`{{define "T"}}{{$}}{{.}}{{end}}{{eval "T" "arg"}}`, "ctx", "argarg"},
		}},
		{"moreThanOneArg", templateTestList{
			{`{{define "T"}}{{$}}{{.}}{{end}}{{eval "T" "a" "b"}}`, "ctx", errors.New("")},
		}},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc.tts.run(t)
		})
	}
}
