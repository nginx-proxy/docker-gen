package template

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func parseAllocateInt(desired string) (int, error) {
	parsed, err := strconv.ParseInt(desired, 10, 32)
	if err != nil {
		return int(0), err
	}
	if parsed < 0 {
		return int(0), fmt.Errorf("non-negative decimal number required for array/slice index, got %#v", desired)
	}
	if parsed <= math.MaxInt32 {
		return int(parsed), nil
	}
	return math.MaxInt32, nil
}

func deepGetImpl(v reflect.Value, path []string) interface{} {
	if !v.IsValid() {
		return nil
	}
	if len(path) == 0 {
		return v.Interface()
	}
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() == reflect.Pointer {
		log.Printf("unable to descend into pointer of a pointer\n")
		return nil
	}
	switch v.Kind() {
	case reflect.Struct:
		return deepGetImpl(v.FieldByName(path[0]), path[1:])
	case reflect.Map:
		// If the first part of the path is a key in the map, we use it directly
		if mapValue := v.MapIndex(reflect.ValueOf(path[0])); mapValue.IsValid() {
			return deepGetImpl(mapValue, path[1:])
		}

		// If the first part of the path is not a key in the map, we try to find a valid key by joining the path parts
		for i := 2; i <= len(path); i++ {
			joinedPath := strings.Join(path[0:i], ".")
			if mapValue := v.MapIndex(reflect.ValueOf(joinedPath)); mapValue.IsValid() {
				if i == len(path) {
					return mapValue.Interface()
				}
				return deepGetImpl(mapValue, path[i:])
			}
		}

		return nil
	case reflect.Slice, reflect.Array:
		i, err := parseAllocateInt(path[0])
		if err != nil {
			log.Println(err.Error())
			return nil
		}
		if i >= v.Len() {
			log.Printf("index %v out of bounds", i)
			return nil
		}
		return deepGetImpl(v.Index(i), path[1:])
	default:
		log.Printf("unable to index by %s (value %v, kind %s)\n", path[0], v, v.Kind())
		return nil
	}
}

func deepGet(item interface{}, path string) interface{} {
	var parts []string
	if path != "" {
		parts = strings.Split(strings.TrimPrefix(path, "."), ".")
	}
	return deepGetImpl(reflect.ValueOf(item), parts)
}
