package services

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func EncodeBase64(input string) string {
	return base64.StdEncoding.EncodeToString([]byte(input))
}

// DecodeBase64 decodes Base64 to text
func DecodeBase64(input string) (string, error) {
	input = strings.TrimSpace(input)

	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(input)
		if err != nil {
			return "", fmt.Errorf("invalid Base64 input: %w", err)
		}
	}

	return string(decoded), nil
}
