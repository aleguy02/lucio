package toon

import (
	"fmt"
	"maps"
)

// Delimiter is the field separator used in inline primitive arrays,
// tabular field lists, and tabular row cells. The TOON spec allows
// comma (default), HTAB, or pipe.
type Delimiter byte

const (
	DelimiterComma Delimiter = ','
	DelimiterTab   Delimiter = '\t'
	DelimiterPipe  Delimiter = '|'
)

func (d Delimiter) valid() bool {
	switch d {
	case DelimiterComma, DelimiterTab, DelimiterPipe:
		return true
	}
	return false
}

// symbol returns the delimiter as it appears inside a bracket segment.
// Comma is the default and is represented as an empty string in headers.
func (d Delimiter) symbol() string {
	if d == DelimiterComma {
		return ""
	}
	return string(d)
}

// EncodeOptions configures the encoder. The zero value is a valid
// default configuration (2-space indent, comma delimiter).
type EncodeOptions struct {
	// Indent is the number of spaces per nesting level. Zero means
	// the default of 2.
	Indent int
	// Delimiter is the document delimiter used for inline primitive
	// arrays and tabular rows. Zero means DelimiterComma.
	Delimiter Delimiter
}

func (o *EncodeOptions) indent() int {
	if o == nil || o.Indent <= 0 {
		return 2
	}
	return o.Indent
}

func (o *EncodeOptions) delim() Delimiter {
	if o == nil || o.Delimiter == 0 {
		return DelimiterComma
	}
	if !o.Delimiter.valid() {
		return DelimiterComma
	}
	return o.Delimiter
}

// DecodeOptions configures the decoder. The zero value is a valid
// default configuration (2-space indent, strict mode).
type DecodeOptions struct {
	// Indent is the expected number of spaces per nesting level.
	// Zero means the default of 2.
	Indent int
	// Strict toggles strict-mode validation per spec §14. The zero
	// value is treated as strict=true; set NonStrict to opt out.
	NonStrict bool
}

func (o *DecodeOptions) indent() int {
	if o == nil || o.Indent <= 0 {
		return 2
	}
	return o.Indent
}

func (o *DecodeOptions) strict() bool {
	if o == nil {
		return true
	}
	return !o.NonStrict
}

// OrderedMap preserves object key insertion order, which TOON requires
// for both encoding and decoding. Setting an existing key updates its
// value without changing position.
type OrderedMap struct {
	keys   []string
	values map[string]any
}

// NewOrderedMap returns an empty OrderedMap.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{values: map[string]any{}}
}

// Set inserts or updates a key. New keys are appended to the order.
func (m *OrderedMap) Set(key string, value any) {
	if m.values == nil {
		m.values = map[string]any{}
	}
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

// Get returns the value for key and whether it was present.
func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.values[key]
	return v, ok
}

// Delete removes a key from the map.
func (m *OrderedMap) Delete(key string) {
	if _, ok := m.values[key]; !ok {
		return
	}
	delete(m.values, key)
	for i, k := range m.keys {
		if k == key {
			m.keys = append(m.keys[:i], m.keys[i+1:]...)
			return
		}
	}
}

// Len returns the number of entries.
func (m *OrderedMap) Len() int { return len(m.keys) }

// Keys returns a copy of the keys in insertion order.
func (m *OrderedMap) Keys() []string {
	out := make([]string, len(m.keys))
	copy(out, m.keys)
	return out
}

// Range iterates over the entries in insertion order. Stop iterating
// by returning false from fn.
func (m *OrderedMap) Range(fn func(key string, value any) bool) {
	for _, k := range m.keys {
		if !fn(k, m.values[k]) {
			return
		}
	}
}

// Map returns the underlying values as a Go map. Order is not
// preserved by the returned map; use Range or Keys for ordered access.
func (m *OrderedMap) Map() map[string]any {
	out := make(map[string]any, len(m.values))
	maps.Copy(out, m.values)
	return out
}

// Error is returned for spec violations during encode/decode.
type Error struct {
	Msg  string
	Line int // 1-based; 0 if not applicable
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("toon: line %d: %s", e.Line, e.Msg)
	}
	return "toon: " + e.Msg
}

func errAt(line int, format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...), Line: line}
}

func errMsg(format string, args ...any) error {
	return &Error{Msg: fmt.Sprintf(format, args...)}
}
