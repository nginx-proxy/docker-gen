package template

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/nginx-proxy/docker-gen/internal/context"
)

// Generalized where function
func generalizedWhere(funcName string, entries interface{}, key string, test func(interface{}) bool) (interface{}, error) {
	entriesVal, err := getArrayValues(funcName, entries)

	if err != nil {
		return nil, err
	}

	selection := make([]interface{}, 0)
	for i := 0; i < entriesVal.Len(); i++ {
		v := entriesVal.Index(i).Interface()

		value := deepGet(v, key)
		if test(value) {
			selection = append(selection, v)
		}
	}

	return selection, nil
}

// selects entries based on key
func where(entries interface{}, key string, cmp interface{}) (interface{}, error) {
	return generalizedWhere("where", entries, key, func(value interface{}) bool {
		return reflect.DeepEqual(value, cmp)
	})
}

// select entries where a key is not equal to a value
func whereNot(entries interface{}, key string, cmp interface{}) (interface{}, error) {
	return generalizedWhere("whereNot", entries, key, func(value interface{}) bool {
		return !reflect.DeepEqual(value, cmp)
	})
}

// selects entries where a key exists
func whereExist(entries interface{}, key string) (interface{}, error) {
	return generalizedWhere("whereExist", entries, key, func(value interface{}) bool {
		return value != nil
	})
}

// selects entries where a key does not exist
func whereNotExist(entries interface{}, key string) (interface{}, error) {
	return generalizedWhere("whereNotExist", entries, key, func(value interface{}) bool {
		return value == nil
	})
}

// selects entries based on key.  Assumes key is delimited and breaks it apart before comparing
func whereAny(entries interface{}, key, sep string, cmp []string) (interface{}, error) {
	return generalizedWhere("whereAny", entries, key, func(value interface{}) bool {
		if value == nil {
			return false
		} else {
			items := strings.Split(value.(string), sep)
			return len(intersect(cmp, items)) > 0
		}
	})
}

// selects entries based on key.  Assumes key is delimited and breaks it apart before comparing
func whereAll(entries interface{}, key, sep string, cmp []string) (interface{}, error) {
	req_count := len(cmp)
	return generalizedWhere("whereAll", entries, key, func(value interface{}) bool {
		if value == nil {
			return false
		} else {
			items := strings.Split(value.(string), sep)
			return len(intersect(cmp, items)) == req_count
		}
	})
}

// generalized whereLabel function
func generalizedWhereLabel(funcName string, containers context.Context, label string, test func(string, bool) bool) (context.Context, error) {
	selection := make([]*context.RuntimeContainer, 0)

	for i := 0; i < len(containers); i++ {
		container := containers[i]

		value, ok := container.Labels[label]
		if test(value, ok) {
			selection = append(selection, container)
		}
	}

	return selection, nil
}

// selects containers that have a particular label
func whereLabelExists(containers context.Context, label string) (context.Context, error) {
	return generalizedWhereLabel("whereLabelExists", containers, label, func(_ string, ok bool) bool {
		return ok
	})
}

// selects containers that have don't have a particular label
func whereLabelDoesNotExist(containers context.Context, label string) (context.Context, error) {
	return generalizedWhereLabel("whereLabelDoesNotExist", containers, label, func(_ string, ok bool) bool {
		return !ok
	})
}

// selects containers with a particular label whose value matches a regular expression
func whereLabelValueMatches(containers context.Context, label, pattern string) (context.Context, error) {
	rx, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return generalizedWhereLabel("whereLabelValueMatches", containers, label, func(value string, ok bool) bool {
		return ok && rx.MatchString(value)
	})
}
