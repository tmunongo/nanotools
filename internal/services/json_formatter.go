package services

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type JSONFormatterOptions struct {
	Indent   int
	SortKeys bool
}

type JSONFormatterResult struct {
	Formatted string
	IsValid   bool
	Error     string
}

func FormatJSON(input string, opts JSONFormatterOptions) JSONFormatterResult {
	input = strings.TrimSpace(input)

	if input == "" {
		return JSONFormatterResult{
			IsValid: false,
			Error:   "Input is empty",
		}
	}

	var data interface{}
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		return JSONFormatterResult{
			IsValid: false,
			Error:   fmt.Sprintf("JSON parse error: %v", err),
		}
	}

	// TODO: fix always sorting keys even when not selected in UI
	if opts.SortKeys {
		data = sortKeys(data)
	}

	indentStr := strings.Repeat(" ", opts.Indent)
	formatted, err := json.MarshalIndent(data, "", indentStr)
	if err != nil {
		return JSONFormatterResult{
			IsValid: false,
			Error:   fmt.Sprintf("Failed to format JSON: %v", err),
		}
	}

	return JSONFormatterResult{
		Formatted: string(formatted),
		IsValid:   true,
	}
}

func sortKeys(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		sorted := make(map[string]interface{})
		for _, k := range keys {
			sorted[k] = sortKeys(v[k])
		}
		return sorted

	case []interface{}:
		for i, item := range v {
			v[i] = sortKeys(item)
		}
		return v

	default:
		return v
	}
}
