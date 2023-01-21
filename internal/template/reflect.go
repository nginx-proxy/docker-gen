package template

import (
	"log"
	"reflect"
	"strings"
)

func stripPrefix(s, prefix string) string {
	path := s
	for {
		if strings.HasPrefix(path, ".") {
			path = path[1:]
			continue
		}
		break
	}
	return path
}

func deepGetImpl(v reflect.Value, path []string) interface{} {
	if !v.IsValid() {
		log.Printf("invalid value\n")
		return nil
	}
	if len(path) == 0 {
		return v.Interface()
	}
	switch v.Kind() {
	case reflect.Struct:
		return deepGetImpl(v.FieldByName(path[0]), path[1:])
	case reflect.Map:
		return deepGetImpl(v.MapIndex(reflect.ValueOf(path[0])), path[1:])
	default:
		log.Printf("unable to index by %s (value %v, kind %s)\n", path[0], v, v.Kind())
		return nil
	}
}

func deepGet(item interface{}, path string) interface{} {
	var parts []string
	if path != "" {
		parts = strings.Split(stripPrefix(path, "."), ".")
	}
	return deepGetImpl(reflect.ValueOf(item), parts)
}
