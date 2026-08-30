package template

import "go.yaml.in/yaml/v3"

// fromYaml decodes YAML into a structured value, ignoring errors.
func fromYaml(v string) any {
	output, _ := mustFromYaml(v)
	return output
}

// mustFromYaml decodes YAML into a structured value, returning errors.
func mustFromYaml(v string) (any, error) {
	var output any
	err := yaml.Unmarshal([]byte(v), &output)
	return output, err
}

// toYaml encodes an item into a YAML string
func toYaml(v any) string {
	output, _ := mustToYaml(v)
	return string(output)
}

// toYaml encodes an item into a YAML string, returning errors
func mustToYaml(v any) (string, error) {
	output, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
