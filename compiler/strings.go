package compiler

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"github.com/NYTimes/openapi2proto/openapi"
)

// since we're not considering unicode here, we're not using unicode.*
func isAlphaNum(r rune) bool {
	return (r >= 0x41 && r <= 0x5a) || // A-Z
		(r >= 0x61 && r <= 0x7a) || // a-z
		(r >= 0x30 && r <= 0x39) // 0-9
}

func allCaps(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		// replace all non-alpha-numeric characters with an underscore
		if !isAlphaNum(r) {
			r = '_'
		} else {
			r = unicode.ToUpper(r)
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func snakeCase(s string) string {
	var buf bytes.Buffer
	var wasUnderscore bool
	for _, r := range s {
		// replace all non-alpha-numeric characters with an underscore
		if !isAlphaNum(r) {
			r = '_'
			wasUnderscore = true
		} else {
			if wasUnderscore {
				r = unicode.ToLower(r)
			}
			wasUnderscore = false
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func camelCase(s string) string {
	var first = true
	var wasUnderscore bool
	var buf bytes.Buffer
	for _, r := range s {
		// replace all non-alpha-numeric characters with an underscore
		if !isAlphaNum(r) {
			r = '_'
		}

		if r == '_' {
			wasUnderscore = true
			continue
		}

		if first || wasUnderscore {
			r = unicode.ToUpper(r)
		}
		first = false
		wasUnderscore = false
		buf.WriteRune(r)
	}

	return buf.String()
}

func cleanSpacing(output []byte) []byte {
	re := regexp.MustCompile(`}\n*message `)
	output = re.ReplaceAll(output, []byte("}\n\nmessage "))
	re = regexp.MustCompile(`}\n*enum `)
	output = re.ReplaceAll(output, []byte("}\n\nenum "))
	re = regexp.MustCompile(`;\n*message `)
	output = re.ReplaceAll(output, []byte(";\n\nmessage "))
	re = regexp.MustCompile(`}\n*service `)
	return re.ReplaceAll(output, []byte("}\n\nservice "))
}

// takes strings like "foo bar baz" and turns it into "foobarbaz"
// if title is true, then "FooBarBaz"
func concatSpaces(s string, title bool) string {
	var buf bytes.Buffer
	var wasSpace bool
	for _, r := range s {
		if unicode.IsSpace(r) {
			wasSpace = true
			continue
		}
		if wasSpace && title {
			r = unicode.ToUpper(r)
		}
		buf.WriteRune(r)
		wasSpace = false
	}
	return buf.String()
}

func cleanAndTitle(s string) string {
	return cleanCharacters(strings.Title(s))
}

func packageName(s string) string {
	return cleanCharacters(strings.ToLower(concatSpaces(s, false)))
}

func serviceName(s string) string {
	return cleanCharacters(concatSpaces(s, true) + "Service")
}

func cleanCharacters(input string) string {
	var buf bytes.Buffer
	for _, r := range input {
		// anything other than a-z, A-Z, 0-9 should be converted
		// to an underscore
		if !isAlphaNum(r) {
			r = '_'
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func compileEndpointName(e *openapi.Endpoint) string {
	return pathMethodToName(e.Path, e.Verb, e.OperationID)
}

func pathMethodToName(path, method, operationID string) string {
	if operationID != "" {
		return OperationIDToName(operationID)
	}

	path = strings.TrimSuffix(path, ".json")
	// Strip query strings. Note that query strings are illegal
	// in swagger paths, but some tooling seems to tolerate them.
	if i := strings.LastIndexByte(path, '?'); i > 0 {
		path = path[:i]
	}

	var buf bytes.Buffer
	for _, r := range path {
		switch r {
		case '_', '-', '.', '/':
			// turn these into spaces
			r = ' '
		case '{', '}', '[', ']', '(', ')':
			// Strip out illegal-for-identifier characters in the path
			// (XXX Shouldn't we be white-listing this instead of
			// removing black-listed characters?)
			continue
		}
		buf.WriteRune(r)
	}

	var name string
	for _, v := range strings.Fields(buf.String()) {
		name += cleanAndTitle(v)
	}
	return cleanAndTitle(method) + name
}

func OperationIDToName(operationID string) string {
	var name string

	operationID = strings.Replace(operationID, "-", " ", -1)
	operationID = strings.Replace(operationID, "_", " ", -1)

	re := regexp.MustCompile(`[\{\}\[\]()/\.]|\?.*`)
	operationID = re.ReplaceAllString(operationID, "")

	for _, n := range strings.Fields(operationID) {
		// ignore trailing "json" suffix
		if strings.ToLower(n) == "json" {
			continue
		}

		if strings.ToUpper(n) == n {
			n = strings.ToLower(n)
		}

		name += cleanAndTitle(n)
	}

	return name
}
