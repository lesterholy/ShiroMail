package mimeheader

import (
	"mime"
	"strings"
)

var decoder = &mime.WordDecoder{}

func Decode(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	decoded, err := decoder.DecodeHeader(trimmed)
	if err != nil {
		return trimmed
	}
	return strings.TrimSpace(decoded)
}

func DecodeValues(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	decoded := make([]string, 0, len(values))
	for _, value := range values {
		decoded = append(decoded, Decode(value))
	}
	return decoded
}

func DecodeMap(headers map[string][]string) map[string][]string {
	if len(headers) == 0 {
		return map[string][]string{}
	}

	decoded := make(map[string][]string, len(headers))
	for key, values := range headers {
		decoded[key] = DecodeValues(values)
	}
	return decoded
}
