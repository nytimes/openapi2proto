package compiler

import (
	"bytes"
	"net/url"
	"path"
	"strings"
	"unicode"

	"github.com/sanposhiho/openapi2proto/openapi"
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

func normalizeFieldName(s string) string {
	var wasUnderscore bool
	var buf bytes.Buffer
	for _, r := range s {
		if !isAlphaNum(r) {
			if !wasUnderscore {
				buf.WriteRune('_')
			}
			wasUnderscore = true
			continue
		}
		wasUnderscore = false
		buf.WriteRune(r)
	}
	return buf.String()
}

func dedupe(s string, r rune) string {
	var buf bytes.Buffer
	var wasTarget bool
	for _, r1 := range s {
		if r1 == r {
			if !wasTarget {
				buf.WriteRune(r1)
				wasTarget = true
			}
			continue
		}

		wasTarget = false
		buf.WriteRune(r1)
	}
	return buf.String()
}

func removeNonAlphaNum(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		if !isAlphaNum(r) {
			switch r {
			case '_', '-', ' ':
				r = '_'
			default:
				continue
			}
		}

		buf.WriteRune(r)
	}
	return buf.String()
}

func snakeCase(s string) string {
	var wasUnderscore bool
	// pass 1: remove all non-alpha-numeric characters EXCEPT for underscore
	s = removeNonAlphaNum(s)
	// pass 2: dedupe '_'
	s = dedupe(s, '_')

	var runes []rune
	for _, r := range s {
		if !isAlphaNum(r) {
			if !wasUnderscore {
				runes = append(runes, '_')
				wasUnderscore = true
			}
			continue
		}
		wasUnderscore = false
		runes = append(runes, r)
	}

	for len(runes) > 0 && runes[0] == '_' {
		runes = runes[1:]
	}
	for len(runes) > 0 && runes[len(runes)-1] == '_' {
		runes = runes[:len(runes)-1]
	}

	// pass 2: for each consecutive upper case characters, insert
	// a break between the last upper case character and its
	// predecessor
	var wasUpper int
	var buf bytes.Buffer
	for i := 0; i < len(runes); i++ {
		if !isAlphaNum(runes[i]) {
			if !wasUnderscore {
				buf.WriteRune('_')
				wasUnderscore = true
			}
			continue
		}

		// if it's upper cased, check if we have a succession of
		// uppercase letters.
		if unicode.IsUpper(runes[i]) {
			if wasUpper == 0 && buf.Len() > 0 && !wasUnderscore {
				buf.WriteRune('_')
			}
			wasUpper++
		} else {
			if wasUpper > 1 && buf.Len() != 1 {
				if len(runes) > 1 && runes[i-2] != '_' && runes[i-1] != '_' {
					buf.Truncate(buf.Len() - 1)                // remove last upper case letter
					buf.WriteRune('_')                         // insert rune
					buf.WriteRune(unicode.ToLower(runes[i-1])) // re-insert last letter
				}
			}
			wasUpper = 0
		}
		wasUnderscore = false
		buf.WriteRune(unicode.ToLower(runes[i]))
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

// takes strings like "foo bar baz" and turns it into "foobarbaz"
// if title is true, then "FooBarBaz"
func concatSpaces(s string, title bool) string {
	var buf bytes.Buffer
	var wasSpace bool
	for i, r := range s {
		if unicode.IsSpace(r) {
			wasSpace = true
			continue
		}
		if i == 0 || (wasSpace && title) {
			r = unicode.ToUpper(r)
		}
		buf.WriteRune(r)
		wasSpace = false
	}
	return buf.String()
}

func packageName(s string) string {
	return cleanCharacters(strings.ToLower(concatSpaces(s, false)))
}

func normalizeServiceName(s string) string {
	return camelCase(concatSpaces(s, true) + "Service")
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

func normalizeEndpointName(e *openapi.Endpoint) string {
	if opID := e.OperationID; len(opID) > 0 {
		return operationIDToName(opID)
	}

	// Strip out file suffix
	p := strings.TrimSuffix(e.Path, path.Ext(e.Path))
	// Strip query strings. Note that query strings are illegal
	// in swagger paths, but some tooling seems to tolerate them.
	if i := strings.LastIndexByte(p, '?'); i > 0 {
		p = p[:i]
	}

	var buf bytes.Buffer
	for _, r := range p {
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

	var name = camelCase(buf.String())
	return camelCase(e.Verb) + name
}

func looksLikeInteger(s string) bool {
	for _, r := range s {
		// charcter should be between "0" to "9" or else.
		if 0x30 > r || 0x39 < r {
			return false
		}
	}
	return true
}

func normalizeEnumName(s string) string {
	s = strings.Replace(s, "&", " AND ", -1)

	// XXX This is a special case for things like
	// N.Y.%20%2F%20Region
	if v, err := url.QueryUnescape(s); err == nil {
		s = v
	}
	s = allCaps(snakeCase(s))
	return s
}

func operationIDToName(s string) string {
	return camelCase(snakeCase(s))
}
