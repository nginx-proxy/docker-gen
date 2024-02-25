package plugin

import "strings"

type KeyFunc[T any, Value any] func(*T) (*Value, error)

// Generalized groupBy function
func GeneralizedGroupBy[T any, Value any](
	entries []*T,
	key KeyFunc[T, Value],
	addEntry func(map[string][]*T, Value, *T),
) (
	map[string][]*T, error,
) {
	groups := make(map[string][]*T)
	for i := 0; i < len(entries); i++ {
		v := entries[i]
		value, err := key(v)
		if err != nil {
			return nil, err
		}
		if value != nil {
			addEntry(groups, *value, v)
		}
	}
	return groups, nil
}

func GroupByMulti[T any](entries []*T, key KeyFunc[T, string], sep string) (map[string][]*T, error) {
	return GeneralizedGroupBy(entries, key, func(groups map[string][]*T, value string, v *T) {
		items := strings.Split(value, sep)
		for _, item := range items {
			groups[item] = append(groups[item], v)
		}
	})
}

// // groupBy groups a generic array or slice by the path property key
// func groupBy(entries interface{}, key string) (map[string][]interface{}, error) {
// 	return generalizedGroupBy("groupBy", entries, key, func(groups map[string][]interface{}, value interface{}, v interface{}) {
// 		groups[value.(string)] = append(groups[value.(string)], v)
// 	})
// }

// // groupByKeys is the same as groupBy but only returns a list of keys
// func groupByKeys(entries interface{}, key string) ([]string, error) {
// 	keys, err := generalizedGroupByKey("groupByKeys", entries, key, func(groups map[string][]interface{}, value interface{}, v interface{}) {
// 		groups[value.(string)] = append(groups[value.(string)], v)
// 	})

// 	if err != nil {
// 		return nil, err
// 	}

// 	ret := []string{}
// 	for k := range keys {
// 		ret = append(ret, k)
// 	}
// 	return ret, nil
// }

// GroupByLabel is the same as groupBy but over a given label
func GroupByLabel(entries []*RuntimeContainer, label string) (map[string][]*RuntimeContainer, error) {
	getLabel := func(container *RuntimeContainer) (*string, error) {
		if value, ok := container.Labels[label]; ok {
			return &value, nil
		}
		return nil, nil
	}
	return GeneralizedGroupBy(entries, getLabel, func(groups map[string][]*RuntimeContainer, value string, v *RuntimeContainer) {
		groups[value] = append(groups[value], v)
	})
}
