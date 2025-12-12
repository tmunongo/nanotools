package services

import (
	"strings"

	"github.com/google/uuid"
)

type UUIDOptions struct {
	Count       int
	Uppercase   bool
	WithHyphens bool
}

func GenerateUUIDs(opts UUIDOptions) []string {
	// Validate count
	if opts.Count < 1 {
		opts.Count = 1
	}
	if opts.Count > 100 {
		opts.Count = 100
	}

	uuids := make([]string, opts.Count)

	for i := 0; i < opts.Count; i++ {
		// Generate a new UUID v4
		id := uuid.New().String()

		// Remove hyphens if requested
		if !opts.WithHyphens {
			id = strings.ReplaceAll(id, "-", "")
		}

		// Convert to uppercase if requested
		if opts.Uppercase {
			id = strings.ToUpper(id)
		}

		uuids[i] = id
	}

	return uuids
}
