package generator

import (
	"strings"

	"github.com/alioygur/godash"
	"golang.org/x/tools/imports"
)

func ToLowerFirstCamelCase(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToLower(string(s[0])) + godash.ToCamelCase(s)[1:]
}

func ToUpperFirst(s string) string {
	if s == "" {
		return s
	}
	if len(s) == 1 {
		return strings.ToLower(string(s[0]))
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func ToLowerSnakeCase(s string) string {
	return strings.ToLower(godash.ToSnakeCase(s))
}

func ToCamelCase(s string) string {
	return godash.ToCamelCase(s)
}

func GoImportsSource(path string, s string) (string, error) {
	is, err := imports.Process(path, []byte(s), nil)
	return string(is), err
}
