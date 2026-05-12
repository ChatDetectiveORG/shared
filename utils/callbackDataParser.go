package utils

import (
	"fmt"
	"strings"
)

// Data froamt: action?arg1=value1&arg2=value2
// Converts callback data to map with arguments
func ParseCallbackData(data string) map[string]string {
	args := strings.Split(data, "?")
	if len(args) < 2 {
		return nil
	}
	argsMap := make(map[string]string)
	for _, arg := range strings.Split(args[1], "&") {
		parts := strings.Split(arg, "=")
		if len(parts) == 2 {
			argsMap[parts[0]] = parts[1]
		}
	}
	return argsMap
}

// Data froamt: action?arg1=value1&arg2=value2
// Converts map with arguments to callback data string
func DumpCallbackData(action string,data map[string]any) string {
	parts := make([]string, 0, len(data))
	for key, value := range data {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	return action + "?" + strings.Join(parts, "&")
}