package toon

import (
	"strconv"
	"strings"
)

// Decode parses a TOON document. Objects are returned as *OrderedMap so
// that key order is preserved.
func Decode(s string, opts *DecodeOptions) (any, error) {
	d := newDecoder(s, opts)
	v, err := d.decodeRoot()
	if err != nil {
		return nil, err
	}
	if d.pos < len(d.lines) {
		return nil, errAt(d.lines[d.pos].num, "unexpected content after document")
	}
	return v, nil
}

type rawLine struct {
	num     int    // 1-based line number
	indent  int    // leading-space count
	content string // trimmed of leading spaces only
}

type decoder struct {
	lines  []rawLine
	pos    int
	indent int
	strict bool
}

func newDecoder(src string, opts *DecodeOptions) *decoder {
	d := &decoder{indent: opts.indent(), strict: opts.strict()}
	rawLines := strings.Split(src, "\n")
	// Trim a single trailing empty line that arises from a final \n.
	if len(rawLines) > 0 && rawLines[len(rawLines)-1] == "" {
		rawLines = rawLines[:len(rawLines)-1]
	}
	for i, raw := range rawLines {
		stripped, indent := stripIndent(raw)
		if strings.TrimSpace(stripped) == "" {
			// Blank line — skipped at the document level. Strict-mode
			// "no blanks inside arrays" is enforced by callers that
			// know whether they're inside an array scope.
			continue
		}
		d.lines = append(d.lines, rawLine{num: i + 1, indent: indent, content: stripped})
	}
	return d
}

func stripIndent(line string) (string, int) {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return line[n:], n
}

// peek returns the next line without consuming it. ok=false at EOF.
func (d *decoder) peek() (rawLine, bool) {
	if d.pos >= len(d.lines) {
		return rawLine{}, false
	}
	return d.lines[d.pos], true
}

func (d *decoder) advance() rawLine {
	l := d.lines[d.pos]
	d.pos++
	return l
}

// depthOf converts a leading-space count to a nesting depth.
func (d *decoder) depthOf(indent int) (int, error) {
	if d.strict && indent%d.indent != 0 {
		return 0, errMsg("indent %d is not a multiple of %d", indent, d.indent)
	}
	return indent / d.indent, nil
}

// decodeRoot performs root-form discovery per §5.
func (d *decoder) decodeRoot() (any, error) {
	if len(d.lines) == 0 {
		return NewOrderedMap(), nil
	}
	first := d.lines[0]
	if first.indent != 0 {
		return nil, errAt(first.num, "root content must start at depth 0")
	}
	// Bare "[]" — empty root array.
	if first.content == "[]" && len(d.lines) == 1 {
		d.pos = 1
		return []any{}, nil
	}
	// Root array header?
	if isRootArrayHeader(first.content) {
		return d.decodeArrayAt(0)
	}
	// Single primitive at root.
	if len(d.lines) == 1 && !containsUnquotedColon(first.content) {
		d.pos = 1
		v, err := parseScalar(first.content)
		if err != nil {
			return nil, errAt(first.num, "%s", err.Error())
		}
		return v, nil
	}
	// Otherwise: object.
	return d.decodeObjectAt(0)
}

// decodeObjectAt parses a sequence of "key: ..." lines at depth d. It
// returns when it encounters a line at depth < d or EOF.
func (d *decoder) decodeObjectAt(depth int) (*OrderedMap, error) {
	obj := NewOrderedMap()
	for {
		line, ok := d.peek()
		if !ok {
			break
		}
		lineDepth, err := d.depthOf(line.indent)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		if lineDepth < depth {
			break
		}
		if lineDepth > depth {
			return nil, errAt(line.num, "unexpected indentation (depth %d, expected %d)", lineDepth, depth)
		}
		// At this depth. List-item markers are not valid object fields.
		if strings.HasPrefix(line.content, "- ") || line.content == "-" {
			break
		}
		d.advance()
		key, rest, err := parseKey(line.content)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		if d.strict {
			if _, dup := obj.Get(key); dup {
				return nil, errAt(line.num, "duplicate key %q", key)
			}
		}
		val, err := d.decodeFieldValue(depth, key, rest, line.num)
		if err != nil {
			return nil, err
		}
		obj.Set(key, val)
	}
	return obj, nil
}

// decodeFieldValue interprets the text following a "key" token. rest
// may be:
//   - "" (no colon present)  → caller error
//   - ":"                    → nested object or empty object
//   - ": value"              → primitive value
//   - "[N…]: …"              → array header (inline or expanded)
//   - "[N…]{fields}: …"      → tabular array header
//   - ": []"                 → empty array
func (d *decoder) decodeFieldValue(depth int, key, rest string, lineNum int) (any, error) {
	// Array header form.
	if len(rest) > 0 && rest[0] == '[' {
		hdr, after, err := parseHeader(rest)
		if err != nil {
			return nil, errAt(lineNum, "%s", err.Error())
		}
		return d.decodeArrayFromHeader(depth, hdr, after, lineNum)
	}
	if rest == "" || rest[0] != ':' {
		return nil, errAt(lineNum, "expected ':' after key %q", key)
	}
	// Skip the colon.
	rest = rest[1:]
	// Bare "key:" → nested or empty object.
	if rest == "" {
		// If the next line is at depth+1, decode object; else empty.
		if next, ok := d.peek(); ok {
			nd, err := d.depthOf(next.indent)
			if err != nil {
				return nil, errAt(next.num, "%s", err.Error())
			}
			if nd == depth+1 {
				return d.decodeObjectAt(depth + 1)
			}
		}
		return NewOrderedMap(), nil
	}
	// Must have a leading space.
	if rest[0] != ' ' {
		return nil, errAt(lineNum, "expected space after ':'")
	}
	value := rest[1:]
	if value == "[]" {
		return []any{}, nil
	}
	return parseScalar(value)
}

// decodeArrayAt parses an array whose header line is the next line at
// depth d.
func (d *decoder) decodeArrayAt(depth int) (any, error) {
	line, ok := d.peek()
	if !ok {
		return nil, errMsg("expected array header at depth %d", depth)
	}
	d.advance()
	hdr, after, err := parseHeader(line.content)
	if err != nil {
		return nil, errAt(line.num, "%s", err.Error())
	}
	return d.decodeArrayFromHeader(depth, hdr, after, line.num)
}

// header represents a parsed `[key]?[N<delim?>]({fields})?:` segment.
type header struct {
	length int
	delim  Delimiter
	fields []string // nil when not tabular
}

// parseHeader parses a header that begins with '[' from s and returns
// the header plus the unparsed remainder after the closing ':'.
func parseHeader(s string) (header, string, error) {
	var h header
	if len(s) == 0 || s[0] != '[' {
		return h, "", errMsg("expected '[' to start array header")
	}
	end := strings.IndexByte(s, ']')
	if end < 0 {
		return h, "", errMsg("unterminated array bracket")
	}
	inner := s[1:end]
	// Strip optional trailing delimiter symbol.
	delim := DelimiterComma
	if len(inner) > 0 {
		last := inner[len(inner)-1]
		if last == '\t' || last == '|' {
			delim = Delimiter(last)
			inner = inner[:len(inner)-1]
		}
	}
	if inner == "" {
		return h, "", errMsg("missing array length")
	}
	if len(inner) > 1 && inner[0] == '0' {
		return h, "", errMsg("invalid array length %q (leading zero)", inner)
	}
	n, err := strconv.Atoi(inner)
	if err != nil || n < 0 {
		return h, "", errMsg("invalid array length %q", inner)
	}
	h.length = n
	h.delim = delim
	rest := s[end+1:]
	// Optional fields segment.
	if len(rest) > 0 && rest[0] == '{' {
		fieldsEnd := strings.IndexByte(rest, '}')
		if fieldsEnd < 0 {
			return h, "", errMsg("unterminated field list")
		}
		fieldsRaw := rest[1:fieldsEnd]
		fields, err := splitFieldNames(fieldsRaw, delim)
		if err != nil {
			return h, "", err
		}
		h.fields = fields
		rest = rest[fieldsEnd+1:]
	}
	if len(rest) == 0 || rest[0] != ':' {
		return h, "", errMsg("expected ':' after array header")
	}
	return h, rest[1:], nil
}

// isRootArrayHeader is a quick check to decide whether a depth-0 line
// looks like a root array header (starts with '[' and follows the
// `[N…]` shape).
func isRootArrayHeader(content string) bool {
	if len(content) == 0 || content[0] != '[' {
		return false
	}
	_, _, err := parseHeader(content)
	return err == nil
}

// decodeArrayFromHeader decodes an array given its already-parsed
// header and the text that followed the header's colon.
func (d *decoder) decodeArrayFromHeader(depth int, h header, afterColon string, lineNum int) (any, error) {
	// Inline form: header line has values after the colon.
	if afterColon != "" {
		if afterColon[0] != ' ' {
			return nil, errAt(lineNum, "expected space after array header colon")
		}
		// Tabular arrays do not put rows on the header line, but if a
		// `{fields}:` header has inline content something is wrong.
		if h.fields != nil {
			return nil, errAt(lineNum, "tabular header cannot have inline values")
		}
		body := afterColon[1:]
		return d.decodeInlinePrimitives(h, body, lineNum)
	}
	// Empty arrays: legacy [0]: form. No body lines follow.
	if h.length == 0 {
		return []any{}, nil
	}
	// Tabular form: read N rows at depth+1.
	if h.fields != nil {
		return d.decodeTabularRows(depth, h)
	}
	// Expanded list form: read N "- …" items at depth+1.
	return d.decodeListItems(depth, h)
}

// decodeInlinePrimitives splits body using h.delim and decodes each
// token as a scalar.
func (d *decoder) decodeInlinePrimitives(h header, body string, lineNum int) ([]any, error) {
	tokens, err := splitDelim(body, h.delim)
	if err != nil {
		return nil, errAt(lineNum, "%s", err.Error())
	}
	if d.strict && len(tokens) != h.length {
		return nil, errAt(lineNum, "inline array length mismatch: header declared %d, got %d", h.length, len(tokens))
	}
	out := make([]any, len(tokens))
	for i, t := range tokens {
		v, err := parseScalar(strings.TrimSpace(t))
		if err != nil {
			return nil, errAt(lineNum, "%s", err.Error())
		}
		out[i] = v
	}
	return out, nil
}

// decodeTabularRows reads h.length rows at depth+1, each split on
// h.delim and matched against h.fields.
func (d *decoder) decodeTabularRows(depth int, h header) ([]any, error) {
	out := make([]any, 0, h.length)
	for {
		line, ok := d.peek()
		if !ok {
			break
		}
		nd, err := d.depthOf(line.indent)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		if nd != depth+1 {
			break
		}
		// Row disambiguation (§9.3): a row has no unquoted colon, or
		// has its first-unquoted delimiter before its first-unquoted
		// colon.
		if !isRowLine(line.content, h.delim) {
			break
		}
		d.advance()
		tokens, err := splitDelim(line.content, h.delim)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		if d.strict && len(tokens) != len(h.fields) {
			return nil, errAt(line.num, "tabular row width %d, expected %d", len(tokens), len(h.fields))
		}
		row := NewOrderedMap()
		for i, f := range h.fields {
			if i >= len(tokens) {
				row.Set(f, "")
				continue
			}
			v, err := parseScalar(strings.TrimSpace(tokens[i]))
			if err != nil {
				return nil, errAt(line.num, "%s", err.Error())
			}
			row.Set(f, v)
		}
		out = append(out, row)
		if len(out) == h.length {
			break
		}
	}
	if d.strict && len(out) != h.length {
		return nil, errMsg("tabular array length mismatch: header declared %d, got %d", h.length, len(out))
	}
	return out, nil
}

// decodeListItems reads h.length expanded list items at depth+1.
func (d *decoder) decodeListItems(depth int, h header) ([]any, error) {
	out := make([]any, 0, h.length)
	for len(out) < h.length {
		line, ok := d.peek()
		if !ok {
			break
		}
		nd, err := d.depthOf(line.indent)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		if nd != depth+1 {
			break
		}
		if line.content != "-" && !strings.HasPrefix(line.content, "- ") {
			break
		}
		item, err := d.decodeListItem(depth+1, h.delim)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if d.strict && len(out) != h.length {
		return nil, errMsg("array length mismatch: header declared %d, got %d", h.length, len(out))
	}
	return out, nil
}

// decodeListItem parses a single "- …" item at the given depth. The
// item's depth refers to the hyphen line; further nesting goes deeper.
// parentDelim is the active delimiter inherited from the parent array
// header (used for inline primitives that follow "- ").
func (d *decoder) decodeListItem(depth int, parentDelim Delimiter) (any, error) {
	line := d.advance()
	content := line.content
	// Empty-object marker.
	if content == "-" {
		// Peek for nested fields at depth+1; if none, it's an empty object.
		if next, ok := d.peek(); ok {
			nd, _ := d.depthOf(next.indent)
			if nd == depth+1 {
				return d.decodeObjectAt(depth + 1)
			}
		}
		return NewOrderedMap(), nil
	}
	// content begins with "- "
	rest := content[2:]
	// Nested inner-array header form: "[M…]: …" or "[M…]:" .
	if len(rest) > 0 && rest[0] == '[' {
		hdr, after, err := parseHeader(rest)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		return d.decodeArrayFromHeader(depth, hdr, after, line.num)
	}
	// Object whose first field is on the hyphen line.
	if containsUnquotedColon(rest) {
		// Parse the first field.
		key, after, err := parseKey(rest)
		if err != nil {
			return nil, errAt(line.num, "%s", err.Error())
		}
		obj := NewOrderedMap()
		val, err := d.decodeFieldValue(depth, key, after, line.num)
		if err != nil {
			return nil, err
		}
		obj.Set(key, val)
		// Remaining fields of this list-item object live at depth+1.
		// Reuse decodeObjectAt by treating subsequent lines as same-depth.
		more, err := d.decodeObjectAt(depth + 1)
		if err != nil {
			return nil, err
		}
		more.Range(func(k string, v any) bool {
			if d.strict {
				if _, dup := obj.Get(k); dup {
					// Mirror duplicate-key behavior; defer to error path.
				}
			}
			obj.Set(k, v)
			return true
		})
		return obj, nil
	}
	// Primitive list item — apply parent's active delimiter for quoting
	// context (parsing scalars doesn't depend on it, but we accept it
	// for symmetry).
	_ = parentDelim
	return parseScalar(strings.TrimSpace(rest))
}

// --- Scalar parsing -------------------------------------------------

// parseScalar decodes a single value token (after any surrounding
// whitespace has been trimmed by the caller).
func parseScalar(tok string) (any, error) {
	if tok == "" {
		return "", nil
	}
	if tok[0] == '"' {
		s, n, err := unquoteString(tok)
		if err != nil {
			return nil, err
		}
		if n != len(tok) {
			return nil, errMsg("unexpected content after quoted string")
		}
		return s, nil
	}
	switch tok {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	}
	if v, ok := tryParseNumber(tok); ok {
		return v, nil
	}
	return tok, nil
}

// tryParseNumber attempts numeric decoding per §4. Tokens with
// forbidden leading zeros are rejected (returned ok=false so they
// decode as strings).
func tryParseNumber(tok string) (any, bool) {
	if tok == "" {
		return nil, false
	}
	i := 0
	if tok[0] == '-' {
		i = 1
	}
	if i >= len(tok) || tok[i] < '0' || tok[i] > '9' {
		return nil, false
	}
	intStart := i
	for i < len(tok) && tok[i] >= '0' && tok[i] <= '9' {
		i++
	}
	intPart := tok[intStart:i]
	hasFrac := false
	if i < len(tok) && tok[i] == '.' {
		hasFrac = true
		i++
		fracStart := i
		for i < len(tok) && tok[i] >= '0' && tok[i] <= '9' {
			i++
		}
		if i == fracStart {
			return nil, false
		}
	}
	hasExp := false
	if i < len(tok) && (tok[i] == 'e' || tok[i] == 'E') {
		hasExp = true
		i++
		if i < len(tok) && (tok[i] == '+' || tok[i] == '-') {
			i++
		}
		expStart := i
		for i < len(tok) && tok[i] >= '0' && tok[i] <= '9' {
			i++
		}
		if i == expStart {
			return nil, false
		}
	}
	if i != len(tok) {
		return nil, false
	}
	// Spec §4: forbidden leading zeros in the integer part (e.g. "05",
	// "0001") decode as strings UNLESS followed by '.' or exponent.
	if len(intPart) > 1 && intPart[0] == '0' && !hasFrac && !hasExp {
		return nil, false
	}
	// Plain "-0" with no fractional/exponent part decodes to 0.
	v, err := strconv.ParseFloat(tok, 64)
	if err != nil {
		return nil, false
	}
	if v == 0 {
		v = 0 // normalize -0 → 0
	}
	return v, true
}

// --- Header / line helpers ------------------------------------------

// parseKey scans a key (quoted or unquoted) from the start of s and
// returns the decoded key and the remainder beginning at the colon.
func parseKey(s string) (string, string, error) {
	if len(s) == 0 {
		return "", "", errMsg("expected key")
	}
	if s[0] == '"' {
		key, n, err := unquoteString(s)
		if err != nil {
			return "", "", err
		}
		return key, s[n:], nil
	}
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ':' || c == '[' || c == ' ' {
			break
		}
		i++
	}
	if i == 0 {
		return "", "", errMsg("missing key")
	}
	return s[:i], s[i:], nil
}

// containsUnquotedColon reports whether s contains a ':' outside of any
// double-quoted substring.
func containsUnquotedColon(s string) bool {
	inQuotes := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' && inQuotes:
			if i+1 < len(s) {
				i++
			}
		case c == '"':
			inQuotes = !inQuotes
		case c == ':' && !inQuotes:
			return true
		}
	}
	return false
}

// isRowLine implements the tabular row disambiguation rule from §9.3.
func isRowLine(s string, delim Delimiter) bool {
	colonIdx := -1
	delimIdx := -1
	inQuotes := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && inQuotes {
			if i+1 < len(s) {
				i++
			}
			continue
		}
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if inQuotes {
			continue
		}
		if colonIdx == -1 && c == ':' {
			colonIdx = i
		}
		if delimIdx == -1 && c == byte(delim) {
			delimIdx = i
		}
		if colonIdx != -1 && delimIdx != -1 {
			break
		}
	}
	if colonIdx == -1 {
		return true
	}
	if delimIdx == -1 {
		return false
	}
	return delimIdx < colonIdx
}

// splitDelim splits s on the given delimiter outside of quoted strings.
func splitDelim(s string, delim Delimiter) ([]string, error) {
	if s == "" {
		// An empty body means zero tokens — used for empty inline arrays.
		return nil, nil
	}
	out := []string{}
	inQuotes := false
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && inQuotes {
			if i+1 < len(s) {
				i++
			}
			continue
		}
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes && c == byte(delim) {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if inQuotes {
		return nil, errMsg("unterminated quoted string in delimited list")
	}
	out = append(out, s[start:])
	return out, nil
}

// splitFieldNames splits a brace-enclosed field list and decodes each
// name (quoted names are unescaped).
func splitFieldNames(s string, delim Delimiter) ([]string, error) {
	if s == "" {
		return nil, errMsg("empty tabular field list")
	}
	parts, err := splitDelim(s, delim)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(parts))
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, errMsg("empty field name")
		}
		if p[0] == '"' {
			k, n, err := unquoteString(p)
			if err != nil {
				return nil, err
			}
			if n != len(p) {
				return nil, errMsg("unexpected content after quoted field name")
			}
			out[i] = k
		} else {
			out[i] = p
		}
	}
	return out, nil
}
