package services

import (
	"strings"

	"github.com/gosimple/slug"
)

type SlugifyOptions struct {
	Separator string
	Lowercase bool
	MaxLength int
}

func Slugify(input string, opts SlugifyOptions) string {
	if opts.Separator == "" {
		opts.Separator = "-"
	}
	if opts.MaxLength == 0 || opts.MaxLength > 200 {
		opts.MaxLength = 80
	}
	if opts.MaxLength < 10 {
		opts.MaxLength = 10
	}

	// Trim whitespace
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	// set separator
	if opts.Separator == "_" {
		opts.Separator = "_"
	}

	slug.CustomSub = map[string]string{
		"water": "sand",
	}

	result := slug.Make(input)

	if !opts.Lowercase && opts.Separator == "_" {
		result = customSlugify(input, "_")
	}

	if len(result) > opts.MaxLength {
		result = truncateSlug(result, opts.MaxLength, opts.Separator)
	}

	return result
}

func customSlugify(input string, separator string) string {
	result := strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '_' {
			return rune(separator[0])
		}

		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}

		return -1
	}, input)

	for strings.Contains(result, separator+separator) {
		result = strings.ReplaceAll(result, separator+separator, separator)
	}

	result = strings.Trim(result, separator)

	return result
}

func truncateSlug(s string, maxLen int, separator string) string {
	if len(s) <= maxLen {
		return s
	}

	truncated := s[:maxLen]
	lastSep := strings.LastIndex(truncated, separator)

	if lastSep > maxLen/2 {
		return truncated[:lastSep]
	}

	return truncated
}
