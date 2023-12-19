package template

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
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
	tmpl.Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
		"closest":                arrayClosest,
		"coalesce":               coalesce,
		"contains":               contains,
		"dir":                    dirList,
		"eval":                   eval,
		"exists":                 utils.PathExists,
		"groupBy":                groupBy,
		"groupByKeys":            groupByKeys,
		"groupByMulti":           groupByMulti,
		"groupByLabel":           groupByLabel,
		"json":                   marshalJson,
		"intersect":              intersect,
		"keys":                   keys,
		"replace":                strings.Replace,
		"parseBool":              strconv.ParseBool,
		"parseJson":              unmarshalJson,
		"queryEscape":            url.QueryEscape,
		"sha1":                   hashSha1,
		"split":                  strings.Split,
		"splitN":                 strings.SplitN,
		"sortStringsAsc":         sortStringsAsc,
		"sortStringsDesc":        sortStringsDesc,
		"sortObjectsByKeysAsc":   sortObjectsByKeysAsc,
		"sortObjectsByKeysDesc":  sortObjectsByKeysDesc,
		"trimPrefix":             trimPrefix,
		"trimSuffix":             trimSuffix,
		"toLower":                toLower,
		"toUpper":                toUpper,
		"when":                   when,
		"where":                  where,
		"whereNot":               whereNot,
		"whereExist":             whereExist,
		"whereNotExist":          whereNotExist,
		"whereAny":               whereAny,
		"whereAll":               whereAll,
		"whereLabelExists":       whereLabelExists,
		"whereLabelDoesNotExist": whereLabelDoesNotExist,
		"whereLabelValueMatches": whereLabelValueMatches,
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
		dest, err := os.CreateTemp(filepath.Dir(config.Dest), "docker-gen")
		defer func() {
			dest.Close()
			os.Remove(dest.Name())
		}()
		if err != nil {
			log.Fatalf("Unable to create temp file: %s\n", err)
		}

		if n, err := dest.Write(contents); n != len(contents) || err != nil {
			log.Fatalf("Failed to write to temp file: wrote %d, exp %d, err=%v", n, len(contents), err)
		}

		oldContents := []byte{}
		if fi, err := os.Stat(config.Dest); err == nil || os.IsNotExist(err) {
			if err != nil && os.IsNotExist(err) {
				emptyFile, err := os.Create(config.Dest)
				if err != nil {
					log.Fatalf("Unable to create empty destination file: %s\n", err)
				} else {
					emptyFile.Close()
					fi, _ = os.Stat(config.Dest)
				}
			}
			if err := dest.Chmod(fi.Mode()); err != nil {
				log.Fatalf("Unable to chmod temp file: %s\n", err)
			}
			if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
				log.Fatalf("Unable to chown temp file: %s\n", err)
			}
			oldContents, err = os.ReadFile(config.Dest)
			if err != nil {
				log.Fatalf("Unable to compare current file contents: %s: %s\n", config.Dest, err)
			}
		}

		if !bytes.Equal(oldContents, contents) {
			err = os.Rename(dest.Name(), config.Dest)
			if err != nil {
				log.Fatalf("Unable to create dest file %s: %s\n", config.Dest, err)
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
	tmpl, err := newTemplate(filepath.Base(templatePath)).ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("Unable to parse template: %s", err)
	}

	buf := new(bytes.Buffer)
	err = tmpl.ExecuteTemplate(buf, filepath.Base(templatePath), &containers)
	if err != nil {
		log.Fatalf("Template error: %s\n", err)
	}
	return buf.Bytes()
}
