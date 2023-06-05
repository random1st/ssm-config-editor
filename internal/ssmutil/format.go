package ssmutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// isValidJSON checks if the given string is valid JSON
func IsValidJSON(data string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(data), &js) == nil
}

// isValidYAML checks if the given string is valid YAML
func IsValidYAML(data string) bool {
	var ym map[string]interface{}
	err := yaml.Unmarshal([]byte(data), &ym)
	fmt.Println("err", err)
	return err == nil
}

// isValidENV checks if the given string is valid ENV format
func IsValidENV(data string) bool {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, "=") {
			return false
		}
	}
	return true
}

func DetectFormat(content []byte) string {
	var format string
	if IsValidJSON(string(content)) {
		format = "json"
	} else if IsValidYAML(string(content)) {
		format = "yaml"
	} else if IsValidENV(string(content)) {
		format = "env"
	} else {
		format = "text"
	}
	fmt.Println("Detected format:", format)
	return format

}

func ValidateFormat(content []byte, format string) error {
	fmt.Println("Validating format:", format)
	switch strings.ToLower(format) {
	case "json":
		if !json.Valid(content) {
			return fmt.Errorf("invalid json")
		}
	case "yaml":
		if !IsValidYAML(string(content)) {
			return fmt.Errorf("invalid yaml")

		}
	case "env":
		if !IsValidENV(string(content)) {
			return fmt.Errorf("invalid env")
		}
	case "text":
		// No format specified, skip validation
		return nil
	default:
		return fmt.Errorf("invalid format")
	}
	return nil
}
