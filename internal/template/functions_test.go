package template

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComment(t *testing.T) {
	env := map[string]string{
		"bar": "baz",
		"foo": "test",
	}

	expected := `# {
#   "bar": "baz",
#   "foo": "test"
# }`

	tests := templateTestList{
		{`{{toPrettyJson . | comment "# "}}`, env, expected},
	}

	tests.run(t)
}

func TestContainsString(t *testing.T) {
	env := map[string]string{
		"PORT": "1234",
	}

	assert.True(t, contains(env, "PORT"))
	assert.False(t, contains(env, "MISSING"))
}

func TestContainsInteger(t *testing.T) {
	env := map[int]int{
		42: 1234,
	}

	assert.True(t, contains(env, 42))
	assert.False(t, contains(env, "WRONG TYPE"))
	assert.False(t, contains(env, 24))
}

func TestContainsNilInput(t *testing.T) {
	var env interface{} = nil

	assert.False(t, contains(env, 0))
	assert.False(t, contains(env, ""))
}

func TestKeys(t *testing.T) {
	env := map[string]string{
		"VIRTUAL_HOST": "demo.local",
	}
	tests := templateTestList{
		{`{{range (keys $)}}{{.}}{{end}}`, env, `VIRTUAL_HOST`},
	}

	tests.run(t)
}

func TestKeysEmpty(t *testing.T) {
	input := map[string]int{}

	k, err := keys(input)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() == reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}

	if len(input) != vk.Len() {
		t.Fatalf("Incorrect key count; expected %d, got %d", len(input), vk.Len())
	}
}

func TestKeysNil(t *testing.T) {
	k, err := keys(nil)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() != reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}
}

func TestInclude(t *testing.T) {
	data := include("some_random_file")
	assert.Equal(t, "", data)

	_ = os.WriteFile("/tmp/docker-gen-test-temp-file", []byte("some string"), 0o777)
	data = include("/tmp/docker-gen-test-temp-file")
	assert.Equal(t, "some string", data)
	_ = os.Remove("/tmp/docker-gen-test-temp-file")
}

func TestIntersect(t *testing.T) {
	i := intersect([]string{"foo.fo.com", "bar.com"}, []string{"foo.bar.com"})
	assert.Len(t, i, 0, "Expected no match")

	i = intersect([]string{"foo.fo.com", "bar.com"}, []string{"bar.com", "foo.com"})
	assert.Len(t, i, 1, "Expected exactly one match")

	i = intersect([]string{"foo.com"}, []string{"bar.com", "foo.com"})
	assert.Len(t, i, 1, "Expected exactly one match")

	i = intersect([]string{"foo.fo.com", "foo.com", "bar.com"}, []string{"bar.com", "foo.com"})
	assert.Len(t, i, 2, "Expected exactly two matches")
}

func TestSplitN(t *testing.T) {
	tests := templateTestList{
		{`{{index (splitN . "/" 2) 0}}`, "example.com/path", `example.com`},
		{`{{index (splitN . "/" 2) 1}}`, "example.com/path", `path`},
		{`{{index (splitN . "/" 2) 1}}`, "example.com/a/longer/path", `a/longer/path`},
		{`{{len (splitN . "/" 2)}}`, "example.com", `1`},
	}

	tests.run(t)
}

func TestParseJson(t *testing.T) {
	tests := templateTestList{
		{`{{parseJson .}}`, `null`, `<no value>`},
		{`{{parseJson .}}`, `true`, `true`},
		{`{{parseJson .}}`, `1`, `1`},
		{`{{parseJson .}}`, `0.5`, `0.5`},
		{`{{index (parseJson .) "enabled"}}`, `{"enabled":true}`, `true`},
		{`{{index (parseJson . | first) "enabled"}}`, `[{"enabled":true}]`, `true`},
	}

	tests.run(t)
}

func TestQueryEscape(t *testing.T) {
	tests := templateTestList{
		{`{{queryEscape .}}`, `example.com`, `example.com`},
		{`{{queryEscape .}}`, `.example.com`, `.example.com`},
		{`{{queryEscape .}}`, `*.example.com`, `%2A.example.com`},
		{`{{queryEscape .}}`, `~^example\.com(\..*\.xip\.io)?$`, `~%5Eexample%5C.com%28%5C..%2A%5C.xip%5C.io%29%3F%24`},
	}

	tests.run(t)
}

func TestArrayClosestExact(t *testing.T) {
	if arrayClosest([]string{"foo.bar.com", "bar.com"}, "foo.bar.com") != "foo.bar.com" {
		t.Fatal("Expected foo.bar.com")
	}
}

func TestArrayClosestSubstring(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bar.com"}, "foo.bar.com") != "bar.com" {
		t.Fatal("Expected bar.com")
	}
}

func TestArrayClosestNoMatch(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bip.com"}, "foo.bar.com") != "" {
		t.Fatal("Expected ''")
	}
}

func TestWhen(t *testing.T) {
	context := struct {
		BoolValue   bool
		StringValue string
	}{
		true,
		"foo",
	}

	tests := templateTestList{
		{`{{ print (when .BoolValue "first" "second") }}`, context, `first`},
		{`{{ print (when (eq .StringValue "foo") "first" "second") }}`, context, `first`},

		{`{{ when (not .BoolValue) "first" "second" | print }}`, context, `second`},
		{`{{ when (not (eq .StringValue "foo")) "first" "second" | print }}`, context, `second`},
	}

	tests.run(t)
}

func TestWhenTrue(t *testing.T) {
	if when(true, "first", "second") != "first" {
		t.Fatal("Expected first value")

	}
}

func TestWhenFalse(t *testing.T) {
	if when(false, "first", "second") != "second" {
		t.Fatal("Expected second value")
	}
}

func TestDirList(t *testing.T) {
	dir, err := os.MkdirTemp("", "dirList")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dir)

	files := map[string]string{
		"aaa": "",
		"bbb": "",
		"ccc": "",
	}
	// Create temporary files
	for key := range files {
		file, err := os.CreateTemp(dir, key)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file.Name())
		files[key] = file.Name()
	}

	expected := []string{
		path.Base(files["aaa"]),
		path.Base(files["bbb"]),
		path.Base(files["ccc"]),
	}

	filesList, _ := dirList(dir)
	assert.Equal(t, expected, filesList)

	filesList, _ = dirList("/wrong/path")
	assert.Equal(t, []string{}, filesList)
}
