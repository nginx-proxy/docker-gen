package dockergen

import (
	"bufio"
	"io"
	"os"
	"strings"
	"unicode"
)

func GetEndpoint(endpoint string) (string, error) {
	defaultEndpoint := "unix:///var/run/docker.sock"
	if os.Getenv("DOCKER_HOST") != "" {
		defaultEndpoint = os.Getenv("DOCKER_HOST")
	}

	if endpoint != "" {
		defaultEndpoint = endpoint
	}

	_, _, err := parseHost(defaultEndpoint)
	if err != nil {
		return "", err
	}

	return defaultEndpoint, nil
}

// splitKeyValueSlice takes a string slice where values are of the form
// KEY, KEY=, KEY=VALUE  or KEY=NESTED_KEY=VALUE2, and returns a map[string]string where items
// are split at their first `=`.
func splitKeyValueSlice(in []string) map[string]string {
	env := make(map[string]string)
	for _, entry := range in {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			parts = append(parts, "")
		}
		env[parts[0]] = parts[1]
	}
	return env

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
