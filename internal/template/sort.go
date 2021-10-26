package template

import (
	"reflect"
	"sort"
)

// sortStrings returns a sorted array of strings in increasing order
func sortStringsAsc(values []string) []string {
	sort.Strings(values)
	return values
}

// sortStringsDesc returns a sorted array of strings in decreasing order
func sortStringsDesc(values []string) []string {
	sort.Sort(sort.Reverse(sort.StringSlice(values)))
	return values
}

type sortable interface {
	sort.Interface
	set(string, interface{}) error
	get() []interface{}
}

type sortableData struct {
	data []interface{}
}

func (s sortableData) get() []interface{} {
	return s.data
}

func (s sortableData) Len() int { return len(s.data) }

func (s sortableData) Swap(i, j int) { s.data[i], s.data[j] = s.data[j], s.data[i] }

type sortableByKey struct {
	sortableData
	key string
}

func (s *sortableByKey) set(funcName string, entries interface{}) (err error) {
	entriesVal, err := getArrayValues(funcName, entries)
	if err != nil {
		return
	}
	s.data = make([]interface{}, entriesVal.Len())
	for i := 0; i < entriesVal.Len(); i++ {
		s.data[i] = reflect.Indirect(entriesVal.Index(i)).Interface()
	}
	return
}

// method required to implement sort.Interface
func (s sortableByKey) Less(i, j int) bool {
	values := map[int]string{i: "", j: ""}
	for k := range values {
		if v := reflect.ValueOf(deepGet(s.data[k], s.key)); v.Kind() != reflect.Invalid {
			values[k] = v.Interface().(string)
		}
	}
	return values[i] < values[j]
}

// Generalized SortBy function
func generalizedSortBy(funcName string, entries interface{}, s sortable, reverse bool) (sorted []interface{}, err error) {
	err = s.set(funcName, entries)
	if err != nil {
		return nil, err
	}
	if reverse {
		sort.Stable(sort.Reverse(s))
	} else {
		sort.Stable(s)
	}
	return s.get(), nil
}

// sortObjectsByKeysAsc returns a sorted array of objects, sorted by object's key field in ascending order
func sortObjectsByKeysAsc(objs interface{}, key string) ([]interface{}, error) {
	s := &sortableByKey{key: key}
	return generalizedSortBy("sortObjsByKeys", objs, s, false)
}

// sortObjectsByKeysDesc returns a sorted array of objects, sorted by object's key field in descending order
func sortObjectsByKeysDesc(objs interface{}, key string) ([]interface{}, error) {
	s := &sortableByKey{key: key}
	return generalizedSortBy("sortObjsByKey", objs, s, true)
}
