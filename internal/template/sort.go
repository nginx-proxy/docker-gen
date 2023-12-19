package template

import (
	"reflect"
	"sort"
	"strconv"
	"time"
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
		s.data[i] = entriesVal.Index(i).Interface()
	}
	return
}

func getFieldAsString(item interface{}, path string) string {
	// Mostly inspired by https://stackoverflow.com/a/47739620
	e := deepGet(item, path)
	r := reflect.ValueOf(e)

	if r.Kind() == reflect.Invalid {
		return ""
	}

	fieldValue := r.Interface()

	switch v := fieldValue.(type) {
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return v.String()
	default:
		return ""
	}
}

// method required to implement sort.Interface
func (s sortableByKey) Less(i, j int) bool {
	dataI := getFieldAsString(s.data[i], s.key)
	dataJ := getFieldAsString(s.data[j], s.key)
	return dataI < dataJ
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
