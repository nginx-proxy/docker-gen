package template

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
)

func keys(input interface{}) (interface{}, error) {
	if input == nil {
		return nil, nil
	}

	val := reflect.ValueOf(input)
	if val.Kind() != reflect.Map {
		return nil, fmt.Errorf("cannot call keys on a non-map value: %v", input)
	}

	vk := val.MapKeys()
	k := make([]interface{}, val.Len())
	for i := range k {
		k[i] = vk[i].Interface()
	}

	return k, nil
}

func include(file string) string {
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(data)
}

func intersect(l1, l2 []string) []string {
	m := make(map[string]bool)
	m2 := make(map[string]bool)
	for _, v := range l2 {
		m2[v] = true
	}
	for _, v := range l1 {
		if m2[v] {
			m[v] = true
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// comment prefix each line of the source string with the provided comment delimiter string
func comment(delimiter string, source string) string {
	regexPattern := regexp.MustCompile(`(?m)^`)
	return regexPattern.ReplaceAllString(source, delimiter)
}

func contains(input interface{}, key interface{}) bool {
	if input == nil {
		return false
	}

	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Map {
		for _, k := range val.MapKeys() {
			if k.Interface() == key {
				return true
			}
		}
	}

	return false
}

// arrayClosest find the longest matching substring in values
// that matches input
func arrayClosest(values []string, input string) string {
	best := ""
	for _, v := range values {
		if strings.Contains(input, v) && len(v) > len(best) {
			best = v
		}
	}
	return best
}

// dirList returns a list of files in the specified path
func dirList(path string) ([]string, error) {
	names := []string{}
	files, err := os.ReadDir(path)
	if err != nil {
		log.Printf("Template error: %v", err)
		return names, nil
	}
	for _, f := range files {
		names = append(names, f.Name())
	}
	return names, nil
}

// when returns the trueValue when the condition is true and the falseValue otherwise
func when(condition bool, trueValue, falseValue interface{}) interface{} {
	if condition {
		return trueValue
	} else {
		return falseValue
	}
}
