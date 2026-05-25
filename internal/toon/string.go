package toon

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// numericLikeRe matches the strings that MUST be quoted because they
// would otherwise decode as numbers (spec §7.2).
var numericLikeRe = regexp.MustCompile(`^-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?$`)

// unquotedKeyRe is the §7.3 unquoted-key pattern.
var unquotedKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)

// quoteString returns s wrapped in double quotes with escapes applied
// per §7.1.
func quoteString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				const hex = "0123456789abcdef"
				b.WriteString(`\u00`)
				b.WriteByte(hex[(r>>4)&0xf])
				b.WriteByte(hex[r&0xf])
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// needsQuoting reports whether s MUST be quoted in a position governed
// by activeDelim (spec §7.2). activeDelim is the delimiter that would
// terminate the value at this position — the document delimiter for
// object-field values, the active delimiter for array cells.
func needsQuoting(s string, activeDelim Delimiter) bool {
	if s == "" {
		return true
	}
	if strings.ContainsAny(s, ":\"\\[]{}") {
		return true
	}
	// Leading or trailing whitespace.
	if s[0] == ' ' || s[len(s)-1] == ' ' ||
		s[0] == '\t' || s[len(s)-1] == '\t' {
		return true
	}
	if s == "true" || s == "false" || s == "null" {
		return true
	}
	if numericLikeRe.MatchString(s) {
		return true
	}
	if s[0] == '-' {
		return true
	}
	if strings.ContainsRune(s, rune(activeDelim)) {
		return true
	}
	for _, r := range s {
		if r < 0x20 {
			return true
		}
	}
	return false
}

// encodeString returns s ready to emit as a TOON string value or
// unquoted-key, choosing quoted vs literal form per the spec.
func encodeString(s string, activeDelim Delimiter) string {
	if needsQuoting(s, activeDelim) {
		return quoteString(s)
	}
	return s
}

// encodeKey returns key formatted for use as an object key or tabular
// field name. Keys are quoted when they fall outside the §7.3 pattern.
func encodeKey(key string) string {
	if unquotedKeyRe.MatchString(key) {
		return key
	}
	return quoteString(key)
}

// unquoteString decodes a quoted TOON string starting at s[0]=='"'.
// Returns the decoded value, the number of bytes consumed (including
// quotes), and any error.
func unquoteString(s string) (string, int, error) {
	if len(s) == 0 || s[0] != '"' {
		return "", 0, errMsg("expected quoted string")
	}
	var b strings.Builder
	i := 1
	for i < len(s) {
		c := s[i]
		if c == '"' {
			return b.String(), i + 1, nil
		}
		if c == '\\' {
			if i+1 >= len(s) {
				return "", 0, errMsg("unterminated escape sequence")
			}
			esc := s[i+1]
			switch esc {
			case '\\':
				b.WriteByte('\\')
				i += 2
			case '"':
				b.WriteByte('"')
				i += 2
			case 'n':
				b.WriteByte('\n')
				i += 2
			case 'r':
				b.WriteByte('\r')
				i += 2
			case 't':
				b.WriteByte('\t')
				i += 2
			case 'u':
				if i+6 > len(s) {
					return "", 0, errMsg("incomplete \\u escape")
				}
				var r rune
				for j := 0; j < 4; j++ {
					ch := s[i+2+j]
					var d rune
					switch {
					case ch >= '0' && ch <= '9':
						d = rune(ch - '0')
					case ch >= 'a' && ch <= 'f':
						d = rune(ch-'a') + 10
					case ch >= 'A' && ch <= 'F':
						d = rune(ch-'A') + 10
					default:
						return "", 0, errMsg("invalid \\u hex digit")
					}
					r = r<<4 | d
				}
				if r >= 0xD800 && r <= 0xDFFF {
					return "", 0, errMsg("lone surrogate in \\u escape")
				}
				b.WriteRune(r)
				i += 6
			default:
				return "", 0, errMsg("invalid escape sequence \\%c", esc)
			}
			continue
		}
		// Literal UTF-8 character.
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return "", 0, errMsg("invalid UTF-8 in quoted string")
		}
		b.WriteRune(r)
		i += size
	}
	return "", 0, errMsg("unterminated quoted string")
}
