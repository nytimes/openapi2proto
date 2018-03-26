package openapi2proto

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"
)

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
		switch {
		case r >= 0x41 && r <= 0x5a: // A-Z
		case r >= 0x61 && r <= 0x7a: // a-z
		case r >= 0x30 && r <= 0x39: // 0-9
		default:
			r = '_'
		}
		buf.WriteRune(r)
	}
	return buf.String()
}
