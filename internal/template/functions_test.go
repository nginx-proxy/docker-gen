package template

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/stretchr/testify/assert"
)

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

	tests.run(t, "keys")
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

func TestHasPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	if !hasPrefix(prefix, str) {
		t.Fatalf("expected %s to have prefix %s", str, prefix)
	}
}

func TestHasSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	if !hasSuffix(suffix, str) {
		t.Fatalf("expected %s to have suffix %s", str, suffix)
	}
}

func TestSplitN(t *testing.T) {
	tests := templateTestList{
		{`{{index (splitN . "/" 2) 0}}`, "example.com/path", `example.com`},
		{`{{index (splitN . "/" 2) 1}}`, "example.com/path", `path`},
		{`{{index (splitN . "/" 2) 1}}`, "example.com/a/longer/path", `a/longer/path`},
		{`{{len (splitN . "/" 2)}}`, "example.com", `1`},
	}

	tests.run(t, "splitN")
}

func TestTrimPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	const trimmed = "127.0.0.1:2375"
	got := trimPrefix(prefix, str)
	if got != trimmed {
		t.Fatalf("expected trimPrefix(%s,%s) to be %s, got %s", prefix, str, trimmed, got)
	}
}

func TestTrimSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	const trimmed = "myhost"
	got := trimSuffix(suffix, str)
	if got != trimmed {
		t.Fatalf("expected trimSuffix(%s,%s) to be %s, got %s", suffix, str, trimmed, got)
	}
}

func TestTrim(t *testing.T) {
	const str = "  myhost.local  "
	const trimmed = "myhost.local"
	got := trim(str)
	if got != trimmed {
		t.Fatalf("expected trim(%s) to be %s, got %s", str, trimmed, got)
	}
}

func TestToLower(t *testing.T) {
	const str = ".RaNd0m StrinG_"
	const lowered = ".rand0m string_"
	assert.Equal(t, lowered, toLower(str), "Unexpected value from toLower()")
}

func TestToUpper(t *testing.T) {
	const str = ".RaNd0m StrinG_"
	const uppered = ".RAND0M STRING_"
	assert.Equal(t, uppered, toUpper(str), "Unexpected value from toUpper()")
}

func TestDict(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	d, err := dict("/", containers)
	if err != nil {
		t.Fatal(err)
	}
	if d["/"] == nil {
		t.Fatalf("did not find containers in dict: %s", d)
	}
	if d["MISSING"] != nil {
		t.Fail()
	}
}

func TestSha1(t *testing.T) {
	sum := hashSha1("/path")
	if sum != "4f26609ad3f5185faaa9edf1e93aa131e2131352" {
		t.Fatal("Incorrect SHA1 sum")
	}
}

func TestJson(t *testing.T) {
	containers := []*context.RuntimeContainer{
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	output, err := marshalJson(containers)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBufferString(output)
	dec := json.NewDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []*context.RuntimeContainer
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(containers) {
		t.Fatalf("Incorrect unmarshaled container count. Expected %d, got %d.", len(containers), len(decoded))
	}
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

	tests.run(t, "parseJson")
}

func TestQueryEscape(t *testing.T) {
	tests := templateTestList{
		{`{{queryEscape .}}`, `example.com`, `example.com`},
		{`{{queryEscape .}}`, `.example.com`, `.example.com`},
		{`{{queryEscape .}}`, `*.example.com`, `%2A.example.com`},
		{`{{queryEscape .}}`, `~^example\.com(\..*\.xip\.io)?$`, `~%5Eexample%5C.com%28%5C..%2A%5C.xip%5C.io%29%3F%24`},
	}

	tests.run(t, "queryEscape")
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

	tests.run(t, "when")
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
	dir, err := ioutil.TempDir("", "dirList")
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
		file, err := ioutil.TempFile(dir, key)
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

func TestCoalesce(t *testing.T) {
	v := coalesce(nil, "second", "third")
	assert.Equal(t, "second", v, "Expected second value")

	v = coalesce(nil, nil, nil)
	assert.Nil(t, v, "Expected nil value")
}
