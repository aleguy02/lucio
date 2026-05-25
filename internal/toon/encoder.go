package toon

import (
	"fmt"
	"reflect"
	"strings"
)

type encoder struct {
	buf       strings.Builder
	indent    int
	delim     Delimiter // document delimiter
	indentStr string
}

func newEncoder(opts *EncodeOptions) *encoder {
	e := &encoder{
		indent: opts.indent(),
		delim:  opts.delim(),
	}
	e.indentStr = strings.Repeat(" ", e.indent)
	return e
}

// Encode serializes v as a TOON document.
func Encode(v any, opts *EncodeOptions) (string, error) {
	e := newEncoder(opts)
	if err := e.encodeRoot(v); err != nil {
		return "", err
	}
	return e.buf.String(), nil
}

func (e *encoder) writeIndent(depth int) {
	for i := 0; i < depth; i++ {
		e.buf.WriteString(e.indentStr)
	}
}

func (e *encoder) encodeRoot(v any) error {
	v = normalize(v)
	switch x := v.(type) {
	case *OrderedMap:
		return e.encodeObjectFields(0, x)
	case []any:
		return e.encodeRootArray(x)
	case nil, bool, string, int64, uint64, float64:
		// Single primitive document.
		e.buf.WriteString(e.formatPrimitiveForField(v))
		return nil
	default:
		return errMsg("cannot encode value of type %T at root", v)
	}
}

// encodeObjectFields writes the fields of an object at the given depth.
// Each field is emitted on its own line; no trailing newline is added
// after the final field.
func (e *encoder) encodeObjectFields(depth int, m *OrderedMap) error {
	keys := m.keys
	for i, k := range keys {
		val, _ := m.Get(k)
		if err := e.encodeField(depth, k, val, i == len(keys)-1); err != nil {
			return err
		}
	}
	return nil
}

// encodeField emits "key: value" / "key:" / a tabular header etc. for a
// single object field. last indicates whether this field is the final
// one being emitted at this depth and so should not end with a newline.
func (e *encoder) encodeField(depth int, key string, val any, last bool) error {
	val = normalize(val)
	encodedKey := encodeKey(key)
	e.writeIndent(depth)
	switch x := val.(type) {
	case *OrderedMap:
		if x.Len() == 0 {
			// Empty object: "key:" on its own line.
			e.buf.WriteString(encodedKey)
			e.buf.WriteString(":")
		} else {
			e.buf.WriteString(encodedKey)
			e.buf.WriteString(":\n")
			if err := e.encodeObjectFields(depth+1, x); err != nil {
				return err
			}
		}
	case []any:
		if err := e.encodeArrayField(depth, encodedKey, x); err != nil {
			return err
		}
	default:
		e.buf.WriteString(encodedKey)
		e.buf.WriteString(": ")
		e.buf.WriteString(e.formatPrimitiveForField(val))
	}
	if !last {
		e.buf.WriteByte('\n')
	}
	return nil
}

// encodeArrayField writes an array as the value of an object field.
// encodedKey is already key-encoded (quoted if needed).
func (e *encoder) encodeArrayField(depth int, encodedKey string, arr []any) error {
	if len(arr) == 0 {
		// Empty arrays use the explicit "key: []" form (§9.1).
		e.buf.WriteString(encodedKey)
		e.buf.WriteString(": []")
		return nil
	}
	// Choose tabular form when all elements are uniform non-empty
	// objects with primitive values (§9.3).
	if fields, ok := tabularFields(arr); ok {
		return e.writeTabular(depth, encodedKey, arr, fields)
	}
	// Decide whether elements are all primitives → inline form, or
	// require an expanded list.
	if allPrimitives(arr) {
		return e.writeInlineArray(encodedKey, arr)
	}
	return e.writeExpandedArray(depth, encodedKey, arr)
}

// encodeRootArray handles the array-at-root special cases (§5, §9).
func (e *encoder) encodeRootArray(arr []any) error {
	if len(arr) == 0 {
		e.buf.WriteString("[]")
		return nil
	}
	if fields, ok := tabularFields(arr); ok {
		return e.writeTabular(0, "", arr, fields)
	}
	if allPrimitives(arr) {
		return e.writeInlineArray("", arr)
	}
	return e.writeExpandedArray(0, "", arr)
}

// writeInlineArray emits `key[N<delim?>]: v1<delim>v2…` (or root form
// when encodedKey is ""). The caller is responsible for any leading
// indentation; no newlines are emitted.
func (e *encoder) writeInlineArray(encodedKey string, arr []any) error {
	d := e.delim
	if encodedKey != "" {
		e.buf.WriteString(encodedKey)
	}
	fmt.Fprintf(&e.buf, "[%d%s]: ", len(arr), d.symbol())
	for i, item := range arr {
		if i > 0 {
			e.buf.WriteByte(byte(d))
		}
		e.buf.WriteString(e.formatPrimitiveForCell(item, d))
	}
	return nil
}

// writeTabular emits a `key[N<delim?>]{f1<delim>f2}:` header followed
// by one row per object at depth+1.
func (e *encoder) writeTabular(depth int, encodedKey string, arr []any, fields []string) error {
	d := e.delim
	if encodedKey != "" {
		e.buf.WriteString(encodedKey)
	}
	fmt.Fprintf(&e.buf, "[%d%s]{", len(arr), d.symbol())
	for i, f := range fields {
		if i > 0 {
			e.buf.WriteByte(byte(d))
		}
		e.buf.WriteString(encodeKey(f))
	}
	e.buf.WriteString("}:")
	for _, item := range arr {
		m, _ := normalize(item).(*OrderedMap)
		e.buf.WriteByte('\n')
		e.writeIndent(depth + 1)
		for i, f := range fields {
			if i > 0 {
				e.buf.WriteByte(byte(d))
			}
			v, _ := m.Get(f)
			e.buf.WriteString(e.formatPrimitiveForCell(v, d))
		}
	}
	return nil
}

// writeExpandedArray emits `key[N<delim?>]:` followed by "- …" items.
func (e *encoder) writeExpandedArray(depth int, encodedKey string, arr []any) error {
	d := e.delim
	if encodedKey != "" {
		e.buf.WriteString(encodedKey)
	}
	fmt.Fprintf(&e.buf, "[%d%s]:", len(arr), d.symbol())
	for _, item := range arr {
		e.buf.WriteByte('\n')
		if err := e.writeListItem(depth+1, item); err != nil {
			return err
		}
	}
	return nil
}

// writeListItem emits a single "- …" list item at the given depth.
func (e *encoder) writeListItem(depth int, item any) error {
	item = normalize(item)
	e.writeIndent(depth)
	switch x := item.(type) {
	case nil, bool, string, int64, uint64, float64:
		e.buf.WriteString("- ")
		// Inside an array scope the active delimiter governs quoting.
		e.buf.WriteString(e.formatPrimitiveForCell(item, e.delim))
	case []any:
		if len(x) == 0 {
			// List-item inner empty array uses legacy header form (§9.2).
			fmt.Fprintf(&e.buf, "- [0%s]:", e.delim.symbol())
			return nil
		}
		if allPrimitives(x) {
			// "- [M<delim?>]: v1<delim>…"
			e.buf.WriteString("- ")
			if err := e.writeInlineArray("", x); err != nil {
				return err
			}
			return nil
		}
		// Nested expanded list inside a list item (§9.4).
		fmt.Fprintf(&e.buf, "- [%d%s]:", len(x), e.delim.symbol())
		for _, sub := range x {
			e.buf.WriteByte('\n')
			if err := e.writeListItem(depth+1, sub); err != nil {
				return err
			}
		}
	case *OrderedMap:
		if x.Len() == 0 {
			// Empty-object list item: bare "-".
			e.buf.WriteByte('-')
			return nil
		}
		return e.writeListItemObject(depth, x)
	default:
		return errMsg("cannot encode list item of type %T", item)
	}
	return nil
}

// writeListItemObject emits an object as a list item per §10. The
// hyphen line is already at the correct indentation; this function
// writes "- " plus the first-field content and any subsequent fields.
func (e *encoder) writeListItemObject(depth int, m *OrderedMap) error {
	keys := m.keys
	firstKey := keys[0]
	firstVal := normalize(m.values[firstKey])
	// Special case: first field is a tabular-eligible array. §10
	// requires the tabular header on the hyphen line, rows at depth+2,
	// other fields at depth+1.
	if arr, ok := firstVal.([]any); ok {
		if fields, isTab := tabularFields(arr); isTab && len(arr) > 0 {
			e.buf.WriteString("- ")
			if err := e.writeTabular(depth+1, encodeKey(firstKey), arr, fields); err != nil {
				return err
			}
			for _, k := range keys[1:] {
				e.buf.WriteByte('\n')
				if err := e.encodeField(depth+1, k, m.values[k], true); err != nil {
					return err
				}
			}
			return nil
		}
	}
	// Otherwise: first field is placed on the hyphen line, remaining
	// fields follow at depth+1.
	e.buf.WriteString("- ")
	if err := e.writeFieldOnHyphen(depth, firstKey, firstVal); err != nil {
		return err
	}
	for _, k := range keys[1:] {
		e.buf.WriteByte('\n')
		if err := e.encodeField(depth+1, k, m.values[k], true); err != nil {
			return err
		}
	}
	return nil
}

// writeFieldOnHyphen writes a single "key: value" / "key:" / array
// header inline (without leading indent) for the first field of a
// list-item object. depth is the depth of the hyphen line; nested
// content uses depth+1.
func (e *encoder) writeFieldOnHyphen(depth int, key string, val any) error {
	val = normalize(val)
	encodedKey := encodeKey(key)
	switch x := val.(type) {
	case *OrderedMap:
		if x.Len() == 0 {
			e.buf.WriteString(encodedKey)
			e.buf.WriteString(":")
		} else {
			e.buf.WriteString(encodedKey)
			e.buf.WriteString(":\n")
			if err := e.encodeObjectFields(depth+1, x); err != nil {
				return err
			}
		}
	case []any:
		if len(x) == 0 {
			e.buf.WriteString(encodedKey)
			e.buf.WriteString(": []")
			return nil
		}
		if fields, ok := tabularFields(x); ok {
			return e.writeTabular(depth+1, encodedKey, x, fields)
		}
		if allPrimitives(x) {
			return e.writeInlineArray(encodedKey, x)
		}
		return e.writeExpandedArray(depth+1, encodedKey, x)
	default:
		e.buf.WriteString(encodedKey)
		e.buf.WriteString(": ")
		e.buf.WriteString(e.formatPrimitiveForField(val))
	}
	return nil
}

// formatPrimitiveForField returns the TOON encoding of a primitive for
// an object-field value, using the document delimiter for quoting (§11.1).
func (e *encoder) formatPrimitiveForField(v any) string {
	return e.formatPrimitive(v, e.delim)
}

// formatPrimitiveForCell returns the TOON encoding of a primitive for
// an inline-array or tabular cell, using activeDelim for quoting.
func (e *encoder) formatPrimitiveForCell(v any, activeDelim Delimiter) string {
	return e.formatPrimitive(v, activeDelim)
}

func (e *encoder) formatPrimitive(v any, activeDelim Delimiter) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case string:
		return encodeString(x, activeDelim)
	case int64:
		return formatInt(x)
	case uint64:
		return formatUint(x)
	case float64:
		return formatFloat(x)
	default:
		// Should not happen after normalize().
		return fmt.Sprintf("%v", v)
	}
}

// tabularFields returns the field list (in encounter order of the first
// element) if arr is a non-empty array of uniform non-empty objects
// with primitive-only values; otherwise ok is false.
func tabularFields(arr []any) (fields []string, ok bool) {
	if len(arr) == 0 {
		return nil, false
	}
	first, isMap := normalize(arr[0]).(*OrderedMap)
	if !isMap || first.Len() == 0 {
		return nil, false
	}
	fields = first.Keys()
	wanted := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		wanted[f] = struct{}{}
	}
	for _, item := range arr {
		m, ok := normalize(item).(*OrderedMap)
		if !ok || m.Len() != len(fields) {
			return nil, false
		}
		for _, k := range m.keys {
			if _, has := wanted[k]; !has {
				return nil, false
			}
			v := normalize(m.values[k])
			if !isPrimitive(v) {
				return nil, false
			}
		}
	}
	return fields, true
}

func allPrimitives(arr []any) bool {
	for _, item := range arr {
		if !isPrimitive(normalize(item)) {
			return false
		}
	}
	return true
}

func isPrimitive(v any) bool {
	switch v.(type) {
	case nil, bool, string, int64, uint64, float64:
		return true
	}
	return false
}

// normalize coerces a Go value into the canonical set of types the
// encoder operates on: nil, bool, string, int64, uint64, float64,
// []any, *OrderedMap. It uses reflection only for the less common
// cases so the fast path (interface assertion) stays cheap.
func normalize(v any) any {
	switch x := v.(type) {
	case nil, bool, string, int64, uint64, float64, []any, *OrderedMap:
		return v
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case uint:
		return uint64(x)
	case uint8:
		return uint64(x)
	case uint16:
		return uint64(x)
	case uint32:
		return uint64(x)
	case float32:
		return float64(x)
	case map[string]any:
		// map[string]any loses key order; preserve declaration order by
		// sorting alphabetically for deterministic output.
		om := NewOrderedMap()
		keys := sortedKeys(x)
		for _, k := range keys {
			om.Set(k, x[k])
		}
		return om
	}
	return normalizeReflect(v)
}

func normalizeReflect(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}
	for rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	case reflect.Map:
		// Generic map: keys must be strings.
		if rv.Type().Key().Kind() != reflect.String {
			return fmt.Sprintf("%v", v)
		}
		om := NewOrderedMap()
		keys := make([]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			keys = append(keys, iter.Key().String())
		}
		// deterministic order
		sortStrings(keys)
		for _, k := range keys {
			om.Set(k, rv.MapIndex(reflect.ValueOf(k)).Interface())
		}
		return om
	case reflect.Struct:
		om := NewOrderedMap()
		t := rv.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := f.Name
			if tag, ok := f.Tag.Lookup("json"); ok && tag != "" {
				parts := strings.SplitN(tag, ",", 2)
				if parts[0] == "-" {
					continue
				}
				if parts[0] != "" {
					name = parts[0]
				}
			}
			om.Set(name, rv.Field(i).Interface())
		}
		return om
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	}
	return fmt.Sprintf("%v", v)
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(s []string) {
	// Tiny insertion sort to avoid pulling in sort package overhead
	// (and to keep this file dependency-light).
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
