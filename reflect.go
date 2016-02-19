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

func joinDots(parts []string) (string, []string) {
	part := parts[0]
	parts = parts[1:]
	for {
		if part[len(part)-1] == '\\' && len(parts) > 0 {
			part = part[0:len(part)-1] + "." + parts[0]
			parts = parts[1:]
			continue
		}
		break
	}
	return part, parts
}

func deepGet(item interface{}, path string) interface{} {
	if path == "" {
		return item
	}

	path = stripPrefix(path, ".")
	parts := strings.Split(path, ".")
	itemValue := reflect.ValueOf(item)

	if len(parts) > 0 {
		part, parts := joinDots(parts)
		switch itemValue.Kind() {
		case reflect.Struct:
			fieldValue := itemValue.FieldByName(part)
			if fieldValue.IsValid() {
				return deepGet(fieldValue.Interface(), strings.Join(parts, "."))
			}
		case reflect.Map:
			mapValue := itemValue.MapIndex(reflect.ValueOf(part))
			if mapValue.IsValid() {
				return deepGet(mapValue.Interface(), strings.Join(parts, "."))
			}
		default:
			log.Printf("can't group by %s (value %v, kind %s)\n", path, itemValue, itemValue.Kind())
		}
		return nil
	}

	return itemValue.Interface()
}
