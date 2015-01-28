package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	env := map[string]string{
		"PORT": "1234",
	}

	if !contains(env, "PORT") {
		t.Fail()
	}

	if contains(env, "MISSING") {
		t.Fail()
	}
}

func TestKeys(t *testing.T) {
	expected := "VIRTUAL_HOST"
	env := map[string]string{
		expected: "demo.local",
	}

	k, err := keys(env)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() == reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}

	if len(env) != vk.Len() {
		t.Fatalf("Incorrect key count; expected %s, got %s", len(env), vk.Len())
	}

	got := vk.Index(0).Interface()
	if expected != got {
		t.Fatalf("Incorrect key found; expected %s, got %s", expected, got)
	}
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
		t.Fatalf("Incorrect key count; expected %s, got %s", len(input), vk.Len())
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
	if len(intersect([]string{"foo.fo.com", "bar.com"}, []string{"foo.bar.com"})) != 0 {
		t.Fatal("Expected no match")
	}

	if len(intersect([]string{"foo.fo.com", "bar.com"}, []string{"bar.com", "foo.com"})) != 1 {
		t.Fatal("Expected only one match")
	}

	if len(intersect([]string{"foo.com"}, []string{"bar.com", "foo.com"})) != 1 {
		t.Fatal("Expected only one match")
	}

	if len(intersect([]string{"foo.fo.com", "foo.com", "bar.com"}, []string{"bar.com", "foo.com"})) != 2 {
		t.Fatal("Expected two matches")
	}
}

func TestGroupByExistingKey(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups := groupBy(containers, "Env.VIRTUAL_HOST")
	if len(groups) != 2 {
		t.Fail()
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fail()
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fail()
	}
	if groups["demo2.localhost"][0].ID != "3" {
		t.Fail()
	}
}

func TestGroupByMulti(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups := groupByMulti(containers, "Env.VIRTUAL_HOST", ",")
	if len(groups) != 3 {
		t.Fatalf("expected 3 got %d", len(groups))
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fatalf("expected 2 got %s", len(groups["demo1.localhost"]))
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fatalf("expected 1 got %s", len(groups["demo2.localhost"]))
	}
	if groups["demo2.localhost"][0].ID != "3" {
		t.Fatalf("expected 2 got %s", groups["demo2.localhost"][0].ID)
	}
	if len(groups["demo3.localhost"]) != 1 {
		t.Fatalf("expect 1 got %d", len(groups["demo3.localhost"]))
	}
	if groups["demo3.localhost"][0].ID != "2" {
		t.Fatalf("expected 2 got %s", groups["demo3.localhost"][0].ID)
	}
}

func TestWhere(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo3.localhost",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	if len(where(containers, "Env.VIRTUAL_HOST", "demo1.localhost")) != 1 {
		t.Fatalf("demo1.localhost expected 1 match")
	}

	if len(where(containers, "Env.VIRTUAL_HOST", "demo2.localhost")) != 2 {
		t.Fatalf("demo2.localhost expected 2 matches")
	}

	if len(where(containers, "Env.VIRTUAL_HOST", "demo3.localhost")) != 1 {
		t.Fatalf("demo3.localhost expected 1 match")
	}

	if len(where(containers, "Env.NOEXIST", "demo3.localhost")) != 0 {
		t.Fatalf("NOEXIST demo3.localhost expected 0 match")
	}
}

func TestWhereSomeMatch(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	if len(whereAny(containers, "Env.VIRTUAL_HOST", ",", []string{"demo1.localhost"})) != 1 {
		t.Fatalf("demo1.localhost expected 1 match")
	}

	if len(whereAny(containers, "Env.VIRTUAL_HOST", ",", []string{"demo2.localhost", "lala"})) != 2 {
		t.Fatalf("demo2.localhost expected 2 matches")
	}

	if len(whereAny(containers, "Env.VIRTUAL_HOST", ",", []string{"something", "demo3.localhost"})) != 1 {
		t.Fatalf("demo3.localhost expected 1 match")
	}

	if len(whereAny(containers, "Env.NOEXIST", ",", []string{"demo3.localhost"})) != 0 {
		t.Fatalf("NOEXIST demo3.localhost expected 0 match")
	}
}

func TestWhereRequires(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost,demo4.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "bar,demo3.localhost,foo",
			},
			ID: "3",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "4",
		},
	}

	if len(whereAll(containers, "Env.VIRTUAL_HOST", ",", []string{"demo1.localhost"})) != 1 {
		t.Fatalf("demo1.localhost expected 1 match")
	}

	if len(whereAll(containers, "Env.VIRTUAL_HOST", ",", []string{"demo2.localhost", "lala"})) != 0 {
		t.Fatalf("demo2.localhost,lala expected 0 matches")
	}

	if len(whereAll(containers, "Env.VIRTUAL_HOST", ",", []string{"demo3.localhost"})) != 1 {
		t.Fatalf("demo3.localhost expected 1 match")
	}

	if len(whereAll(containers, "Env.NOEXIST", ",", []string{"demo3.localhost"})) != 0 {
		t.Fatalf("NOEXIST demo3.localhost expected 0 match")
	}
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

func TestDict(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
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
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
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
	var decoded []*RuntimeContainer
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(containers) {
		t.Fatal("Incorrect unmarshaled container count. Expected %d, got %d.", len(containers), len(decoded))
	}
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
