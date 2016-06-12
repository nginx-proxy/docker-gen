package dockergen

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

func deepGet(item interface{}, path string) interface{} {
	if path == "" {
		return item
	}

	path = stripPrefix(path, ".")
	parts := strings.Split(path, ".")
	itemValue := reflect.ValueOf(item)

	if len(parts) > 0 {
		switch itemValue.Kind() {
		case reflect.Struct:
			fieldValue := itemValue.FieldByName(parts[0])
			if fieldValue.IsValid() {
				return deepGet(fieldValue.Interface(), strings.Join(parts[1:], "."))
			}
		case reflect.Map:
			mapValue := itemValue.MapIndex(reflect.ValueOf(parts[0]))
			if mapValue.IsValid() {
				return deepGet(mapValue.Interface(), strings.Join(parts[1:], "."))
			}
		default:
			log.Printf("Can't group by %s (value %v, kind %s)\n", path, itemValue, itemValue.Kind())
		}
		return nil
	}

	return itemValue.Interface()
}
