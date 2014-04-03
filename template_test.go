package main

import (
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

func TestGroupByExitingKey(t *testing.T) {
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
