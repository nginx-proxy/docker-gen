package template

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/nginx-proxy/docker-gen/internal/config"
	"github.com/nginx-proxy/docker-gen/internal/context"
	"github.com/nginx-proxy/docker-gen/internal/utils"
)

func getArrayValues(funcName string, entries interface{}) (*reflect.Value, error) {
	entriesVal := reflect.ValueOf(entries)

	kind := entriesVal.Kind()

	if kind == reflect.Ptr {
		entriesVal = entriesVal.Elem()
		kind = entriesVal.Kind()
	}

	switch kind {
	case reflect.Array, reflect.Slice:
		break
	default:
		return nil, fmt.Errorf("must pass an array or slice to '%v'; received %v; kind %v", funcName, entries, kind)
	}
	return &entriesVal, nil
}

func newTemplate(name string) *template.Template {
	tmpl := template.New(name)
	// The eval function is defined here because it must be a closure around tmpl.
	eval := func(name string, args ...any) (string, error) {
		buf := bytes.NewBuffer(nil)
		data := any(nil)
		if len(args) == 1 {
			data = args[0]
		} else if len(args) > 1 {
			return "", errors.New("too many arguments")
		}
		if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	sprigFuncMap := sprig.TxtFuncMap()

	tmpl.Funcs(sprigFuncMap).Funcs(template.FuncMap{
		"closest":                 arrayClosest,
		"coalesce":                coalesce,
		"comment":                 comment,
		"contains":                contains,
		"dir":                     dirList,
		"eval":                    eval,
		"exists":                  utils.PathExists,
		"groupBy":                 groupBy,
		"groupByWithDefault":      groupByWithDefault,
		"groupByKeys":             groupByKeys,
		"groupByMulti":            groupByMulti,
		"groupByLabel":            groupByLabel,
		"groupByLabelWithDefault": groupByLabelWithDefault,
		"include":                 include,
		"intersect":               intersect,
		"keys":                    keys,
		"replace":                 strings.Replace,
		"parseBool":               strconv.ParseBool,
		"fromYaml":                fromYaml,
		"toYaml":                  toYaml,
		"mustFromYaml":            mustFromYaml,
		"mustToYaml":              mustToYaml,
		"queryEscape":             url.QueryEscape,
		"split":                   strings.Split,
		"splitN":                  strings.SplitN,
		"sortStringsAsc":          sortStringsAsc,
		"sortStringsDesc":         sortStringsDesc,
		"sortObjectsByKeysAsc":    sortObjectsByKeysAsc,
		"sortObjectsByKeysDesc":   sortObjectsByKeysDesc,
		"toLower":                 strings.ToLower,
		"toUpper":                 strings.ToUpper,
		"when":                    when,
		"where":                   where,
		"whereNot":                whereNot,
		"whereExist":              whereExist,
		"whereNotExist":           whereNotExist,
		"whereAny":                whereAny,
		"whereAll":                whereAll,
		"whereLabelExists":        whereLabelExists,
		"whereLabelDoesNotExist":  whereLabelDoesNotExist,
		"whereLabelValueMatches":  whereLabelValueMatches,

		// legacy docker-gen template function aliased to their Sprig clone
		"json":      sprigFuncMap["mustToJson"],
		"parseJson": sprigFuncMap["mustFromJson"],
		"sha1":      sprigFuncMap["sha1sum"],

		// aliases to sprig template functions masked by docker-gen functions with the same name
		"sprigCoalesce": sprigFuncMap["coalesce"],
		"sprigContains": sprigFuncMap["contains"],
		"sprigDir":      sprigFuncMap["dir"],
		"sprigReplace":  sprigFuncMap["replace"],
		"sprigSplit":    sprigFuncMap["split"],
		"sprigSplitn":   sprigFuncMap["splitn"],
	})

	return tmpl
}

func isBlank(str string) bool {
	for _, r := range str {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func removeBlankLines(reader io.Reader, writer io.Writer) {
	breader := bufio.NewReader(reader)
	bwriter := bufio.NewWriter(writer)

	for {
		line, err := breader.ReadString('\n')

		if !isBlank(line) {
			bwriter.WriteString(line)
		}

		if err != nil {
			break
		}
	}

	bwriter.Flush()
}

func filterRunning(config config.Config, containers context.Context) context.Context {
	if config.IncludeStopped {
		return containers
	} else {
		filteredContainers := context.Context{}
		for _, container := range containers {
			if container.State.Running {
				filteredContainers = append(filteredContainers, container)
			}
		}
		return filteredContainers
	}
}

func GenerateFile(config config.Config, containers context.Context) bool {
	filteredRunningContainers := filterRunning(config, containers)
	filteredContainers := context.Context{}
	if config.OnlyPublished {
		for _, container := range filteredRunningContainers {
			if len(container.PublishedAddresses()) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else if config.OnlyExposed {
		for _, container := range filteredRunningContainers {
			if len(container.Addresses) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else {
		filteredContainers = filteredRunningContainers
	}

	contents := executeTemplate(config.Template, filteredContainers)

	if !config.KeepBlankLines {
		buf := new(bytes.Buffer)
		removeBlankLines(bytes.NewReader(contents), buf)
		contents = buf.Bytes()
	}

	if config.Dest != "" {
		oldContents, err := os.ReadFile(config.Dest)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("Unable to compare current file contents: %s: %s\n", config.Dest, err)
		}

		if !bytes.Equal(oldContents, contents) {
			err := os.WriteFile(config.Dest, contents, 0644)
			if err != nil {
				log.Fatalf("Unable to write to dest file %s: %s\n", config.Dest, err)
			}

			log.Printf("Generated '%s' from %d containers", config.Dest, len(filteredContainers))
			return true
		}
		return false
	} else {
		os.Stdout.Write(contents)
	}
	return true
}

func executeTemplate(templatePath string, containers context.Context) []byte {
	templatePathList := strings.Split(templatePath, ";")
	tmpl, err := newTemplate(filepath.Base(templatePath)).ParseFiles(templatePathList...)
	if err != nil {
		log.Fatalf("Unable to parse template: %s", err)
	}

	buf := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(buf, filepath.Base(templatePathList[0]), &containers)
	if err != nil {
		log.Fatalf("Template error: %s\n", err)
	}
	return buf.Bytes()
}
