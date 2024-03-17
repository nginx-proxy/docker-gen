package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testYaml = `bool: true
list:
    - foo
    - bar
number: 42
string: test
`

var testJson = `{"bool":true,"list":["foo","bar"],"number":42,"string":"test"}`

var testDict = map[string]interface{}{
	"bool":   true,
	"number": 42,
	"string": "test",
	"list": []interface{}{
		"foo",
		"bar",
	},
}

func TestFromYaml(t *testing.T) {
	assert.Equal(t, testDict, fromYaml(testYaml))
	assert.Equal(t, testDict, fromYaml(testJson))
}

func TestToYaml(t *testing.T) {
	assert.Equal(t, testYaml, toYaml(testDict))
}
