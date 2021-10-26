package template

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

type templateTestList []struct {
	tmpl     string
	context  interface{}
	expected string
}

func (tests templateTestList) run(t *testing.T, prefix string) {
	for n, test := range tests {
		tmplName := fmt.Sprintf("%s-test-%d", prefix, n)
		tmpl := template.Must(newTemplate(tmplName).Parse(test.tmpl))

		var b bytes.Buffer
		err := tmpl.ExecuteTemplate(&b, tmplName, test.context)
		if err != nil {
			t.Fatalf("Error executing template: %v (test %s)", err, tmplName)
		}

		got := b.String()
		if test.expected != got {
			t.Fatalf("Incorrect output found; expected %s, got %s (test %s)", test.expected, got, tmplName)
		}
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
