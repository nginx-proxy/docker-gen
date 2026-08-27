package template

import (
	"fmt"
	"strings"

	"github.com/nginx-proxy/docker-gen/internal/context"
)

// Generalized groupBy function
func generalizedGroupBy(funcName string, entries any, getValue func(any) (any, error), addEntry func(map[string][]any, any, any)) (map[string][]any, error) {
	entriesVal, err := getArrayValues(funcName, entries)

	if err != nil {
		return nil, err
	}

	groups := make(map[string][]any)
	for i := 0; i < entriesVal.Len(); i++ {
		v := entriesVal.Index(i).Interface()
		value, err := getValue(v)
		if err != nil {
			return nil, err
		}
		if value != nil {
			addEntry(groups, value, v)
		}
	}
	return groups, nil
}

func generalizedGroupByKey(funcName string, entries any, key string, addEntry func(map[string][]any, any, any)) (map[string][]any, error) {
	getKey := func(v any) (any, error) {
		return deepGet(v, key), nil
	}
	return generalizedGroupBy(funcName, entries, getKey, addEntry)
}

func groupByMulti(entries any, key, sep string) (map[string][]any, error) {
	return generalizedGroupByKey("groupByMulti", entries, key, func(groups map[string][]any, value any, v any) {
		items := strings.SplitSeq(value.(string), sep)
		for item := range items {
			groups[item] = append(groups[item], v)
		}
	})
}

// groupBy groups a generic array or slice by the path property key
func groupBy(entries any, key string) (map[string][]any, error) {
	return generalizedGroupByKey("groupBy", entries, key, func(groups map[string][]any, value any, v any) {
		groups[value.(string)] = append(groups[value.(string)], v)
	})
}

// groupByWithDefault is the same as groupBy but allows a default value to be set
func groupByWithDefault(entries any, key string, defaultValue string) (map[string][]any, error) {
	getValueWithDefault := func(v any) (any, error) {
		value := deepGet(v, key)
		if value == nil {
			return defaultValue, nil
		}
		return value, nil
	}
	return generalizedGroupBy("groupByWithDefault", entries, getValueWithDefault, func(groups map[string][]any, value any, v any) {
		groups[value.(string)] = append(groups[value.(string)], v)
	})
}

// groupByKeys is the same as groupBy but only returns a list of keys
func groupByKeys(entries any, key string) ([]string, error) {
	keys, err := generalizedGroupByKey("groupByKeys", entries, key, func(groups map[string][]any, value any, v any) {
		groups[value.(string)] = append(groups[value.(string)], v)
	})

	if err != nil {
		return nil, err
	}

	ret := []string{}
	for k := range keys {
		ret = append(ret, k)
	}
	return ret, nil
}

// groupByLabel is the same as groupBy but over a given label
func groupByLabel(entries any, label string) (map[string][]any, error) {
	getLabel := func(v any) (any, error) {
		if container, ok := v.(*context.RuntimeContainer); ok {
			if value, ok := container.Labels[label]; ok {
				return value, nil
			}
			return nil, nil
		}
		return nil, fmt.Errorf("must pass an array or slice of *RuntimeContainer to 'groupByLabel'; received %v", v)
	}
	return generalizedGroupBy("groupByLabel", entries, getLabel, func(groups map[string][]any, value any, v any) {
		groups[value.(string)] = append(groups[value.(string)], v)
	})
}

// groupByLabelWithDefault is the same as groupByLabel but allows a default value to be set
func groupByLabelWithDefault(entries any, label string, defaultValue string) (map[string][]any, error) {
	getLabel := func(v any) (any, error) {
		if container, ok := v.(*context.RuntimeContainer); ok {
			if value, ok := container.Labels[label]; ok {
				return value, nil
			}
			return defaultValue, nil
		}
		return nil, fmt.Errorf("must pass an array or slice of *RuntimeContainer to 'groupByLabel'; received %v", v)
	}
	return generalizedGroupBy("groupByLabelWithDefault", entries, getLabel, func(groups map[string][]any, value any, v any) {
		groups[value.(string)] = append(groups[value.(string)], v)
	})
}
