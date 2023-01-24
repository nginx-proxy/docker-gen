package template

import (
	"log"
	"math"
	"reflect"
	"strconv"
	"strings"
)

func deepGetImpl(v reflect.Value, path []string) interface{} {
	if !v.IsValid() {
		log.Printf("invalid value\n")
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
		return deepGetImpl(v.MapIndex(reflect.ValueOf(path[0])), path[1:])
	case reflect.Slice, reflect.Array:
		iu64, err := strconv.ParseUint(path[0], 10, 64)
		if err != nil {
			log.Printf("non-negative decimal number required for array/slice index, got %#v\n", path[0])
			return nil
		}
		if iu64 > math.MaxInt {
			iu64 = math.MaxInt
		}
		i := int(iu64)
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
