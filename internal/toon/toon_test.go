package toon

import (
	"math"
	"strings"
	"testing"
)

// om is a tiny helper for building expected OrderedMaps in tests.
func om(pairs ...any) *OrderedMap {
	m := NewOrderedMap()
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i].(string), pairs[i+1])
	}
	return m
}

func mustEncode(t *testing.T, v any, opts *EncodeOptions) string {
	t.Helper()
	out, err := Encode(v, opts)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return out
}

func mustDecode(t *testing.T, s string, opts *DecodeOptions) any {
	t.Helper()
	v, err := Decode(s, opts)
	if err != nil {
		t.Fatalf("Decode: %v\ninput:\n%s", err, s)
	}
	return v
}

func TestEncodePrimitives(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"null", nil, "null"},
		{"true", true, "true"},
		{"false", false, "false"},
		{"int", 42, "42"},
		{"negative", -7, "-7"},
		{"float", 3.14, "3.14"},
		{"float-integer", 1.0, "1"},
		{"neg-zero", math.Copysign(0, -1), "0"},
		{"string-simple", "hello", "hello"},
		{"string-needs-quote-colon", "a:b", `"a:b"`},
		{"string-numeric-like", "42", `"42"`},
		{"string-true-literal", "true", `"true"`},
		{"empty-string", "", `""`},
		{"large-float-exponent", 1e21, "1e+21"},
		{"small-float-exponent", 1e-7, "1e-07"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mustEncode(t, c.in, nil)
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestEncodeObject(t *testing.T) {
	in := om("id", 123, "name", "Ada", "active", true)
	want := "id: 123\nname: Ada\nactive: true"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeNestedObject(t *testing.T) {
	in := om("user", om("id", 123, "name", "Ada"))
	want := "user:\n  id: 123\n  name: Ada"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeInlinePrimitiveArray(t *testing.T) {
	in := om("tags", []any{"admin", "ops", "dev"})
	want := "tags[3]: admin,ops,dev"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncodeTabularArray(t *testing.T) {
	in := om("items", []any{
		om("sku", "A1", "qty", 2, "price", 9.99),
		om("sku", "B2", "qty", 1, "price", 14.5),
	})
	want := "items[2]{sku,qty,price}:\n  A1,2,9.99\n  B2,1,14.5"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeArrayOfArrays(t *testing.T) {
	in := om("pairs", []any{
		[]any{1, 2},
		[]any{3, 4},
	})
	want := "pairs[2]:\n  - [2]: 1,2\n  - [2]: 3,4"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeEmpty(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"empty-obj-field", om("x", NewOrderedMap()), "x:"},
		{"empty-arr-field", om("x", []any{}), "x: []"},
		{"empty-root-obj", NewOrderedMap(), ""},
		{"empty-root-arr", []any{}, "[]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mustEncode(t, c.in, nil)
			if got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestEncodeTabDelimiter(t *testing.T) {
	in := om("items", []any{
		om("a", 1, "b", 2),
		om("a", 3, "b", 4),
	})
	opts := &EncodeOptions{Delimiter: DelimiterTab}
	want := "items[2\t]{a\tb}:\n  1\t2\n  3\t4"
	got := mustEncode(t, in, opts)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncodePipeDelimiter(t *testing.T) {
	in := om("xs", []any{"a", "b,c", "d"})
	opts := &EncodeOptions{Delimiter: DelimiterPipe}
	want := "xs[3|]: a|b,c|d"
	got := mustEncode(t, in, opts)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncodeMixedArray(t *testing.T) {
	in := om("xs", []any{1, "two", om("k", "v")})
	want := "xs[3]:\n  - 1\n  - two\n  - k: v"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeListItemTabularFirst(t *testing.T) {
	// §10: when a list-item object's first field is a tabular array,
	// the header goes on the hyphen line, rows at depth+2, other
	// fields at depth+1.
	in := om("groups", []any{
		om(
			"items", []any{om("a", 1), om("a", 2)},
			"name", "first",
		),
	})
	want := "groups[1]:\n  - items[2]{a}:\n      1\n      2\n    name: first"
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeQuotesSpecialStrings(t *testing.T) {
	in := om("v", "needs \"quotes\" and \\backslash")
	want := `v: "needs \"quotes\" and \\backslash"`
	got := mustEncode(t, in, nil)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDecodePrimitives(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"hello", "hello"},
		{"42", 42.0},
		{"true", true},
		{"false", false},
		{"null", nil},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := mustDecode(t, c.in, nil)
			if got != c.want {
				t.Fatalf("got %#v want %#v", got, c.want)
			}
		})
	}
}

func TestDecodeObject(t *testing.T) {
	in := "id: 123\nname: Ada\nactive: true"
	v := mustDecode(t, in, nil)
	m, ok := v.(*OrderedMap)
	if !ok {
		t.Fatalf("want *OrderedMap, got %T", v)
	}
	if id, _ := m.Get("id"); id != 123.0 {
		t.Errorf("id: got %#v", id)
	}
	if name, _ := m.Get("name"); name != "Ada" {
		t.Errorf("name: got %#v", name)
	}
	if active, _ := m.Get("active"); active != true {
		t.Errorf("active: got %#v", active)
	}
	if keys := m.Keys(); !equalStrings(keys, []string{"id", "name", "active"}) {
		t.Errorf("key order: %v", keys)
	}
}

func TestDecodeNested(t *testing.T) {
	in := "user:\n  id: 1\n  name: Bob"
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	u, _ := m.Get("user")
	um := u.(*OrderedMap)
	if id, _ := um.Get("id"); id != 1.0 {
		t.Errorf("nested id: %#v", id)
	}
}

func TestDecodeInlineArray(t *testing.T) {
	in := "tags[3]: admin,ops,dev"
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	tags, _ := m.Get("tags")
	arr, ok := tags.([]any)
	if !ok || len(arr) != 3 {
		t.Fatalf("tags: %#v", tags)
	}
	if arr[0] != "admin" || arr[1] != "ops" || arr[2] != "dev" {
		t.Errorf("tags: %#v", arr)
	}
}

func TestDecodeTabular(t *testing.T) {
	in := "items[2]{sku,qty,price}:\n  A1,2,9.99\n  B2,1,14.5"
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	items, _ := m.Get("items")
	arr := items.([]any)
	if len(arr) != 2 {
		t.Fatalf("want 2 rows, got %d", len(arr))
	}
	r0 := arr[0].(*OrderedMap)
	if sku, _ := r0.Get("sku"); sku != "A1" {
		t.Errorf("sku: %#v", sku)
	}
	if qty, _ := r0.Get("qty"); qty != 2.0 {
		t.Errorf("qty: %#v", qty)
	}
	if p, _ := r0.Get("price"); p != 9.99 {
		t.Errorf("price: %#v", p)
	}
}

func TestDecodeRootArray(t *testing.T) {
	in := "[2]{a,b}:\n  1,2\n  3,4"
	v := mustDecode(t, in, nil)
	arr, ok := v.([]any)
	if !ok || len(arr) != 2 {
		t.Fatalf("root array: %#v", v)
	}
}

func TestDecodeEmptyArrays(t *testing.T) {
	for _, src := range []string{"x: []", "[]"} {
		t.Run(src, func(t *testing.T) {
			v := mustDecode(t, src, nil)
			if src == "[]" {
				if arr, ok := v.([]any); !ok || len(arr) != 0 {
					t.Fatalf("got %#v", v)
				}
				return
			}
			m := v.(*OrderedMap)
			x, _ := m.Get("x")
			if arr, ok := x.([]any); !ok || len(arr) != 0 {
				t.Fatalf("got %#v", x)
			}
		})
	}
}

func TestDecodeQuotedNumeric(t *testing.T) {
	// "42" must remain a string (numeric-like, quoted).
	in := `v: "42"`
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	got, _ := m.Get("v")
	if got != "42" {
		t.Fatalf("got %#v", got)
	}
}

func TestStrictArrayLengthMismatch(t *testing.T) {
	in := "x[3]: 1,2"
	_, err := Decode(in, nil)
	if err == nil {
		t.Fatal("expected length mismatch error")
	}
	if !strings.Contains(err.Error(), "length mismatch") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNonStrictAcceptsMismatch(t *testing.T) {
	in := "x[3]: 1,2"
	v, err := Decode(in, &DecodeOptions{NonStrict: true})
	if err != nil {
		t.Fatalf("non-strict: %v", err)
	}
	m := v.(*OrderedMap)
	x, _ := m.Get("x")
	arr := x.([]any)
	if len(arr) != 2 {
		t.Errorf("want 2 elements, got %d", len(arr))
	}
}

func TestRoundTrip(t *testing.T) {
	original := om(
		"context", om(
			"task", "Our favorite hikes together",
			"location", "Boulder",
			"season", "spring_2025",
		),
		"friends", []any{"ana", "luis", "sam"},
		"hikes", []any{
			om("id", 1, "name", "Blue Lake Trail", "distanceKm", 7.5, "elevationGain", 320, "companion", "ana", "wasSunny", true),
			om("id", 2, "name", "Ridge Overlook", "distanceKm", 9.2, "elevationGain", 540, "companion", "luis", "wasSunny", false),
			om("id", 3, "name", "Wildflower Loop", "distanceKm", 5.1, "elevationGain", 180, "companion", "sam", "wasSunny", true),
		},
	)
	encoded := mustEncode(t, original, nil)
	decoded := mustDecode(t, encoded, nil)
	reEncoded := mustEncode(t, decoded, nil)
	if encoded != reEncoded {
		t.Fatalf("round-trip drift\nfirst:\n%s\nsecond:\n%s", encoded, reEncoded)
	}
}

func TestRoundTripWithDelimiters(t *testing.T) {
	src := om("xs", []any{"a,b", "c", "d"})
	for _, d := range []Delimiter{DelimiterComma, DelimiterTab, DelimiterPipe} {
		t.Run(string(d), func(t *testing.T) {
			enc := mustEncode(t, src, &EncodeOptions{Delimiter: d})
			dec := mustDecode(t, enc, nil)
			reEnc := mustEncode(t, dec, &EncodeOptions{Delimiter: d})
			if enc != reEnc {
				t.Fatalf("drift\n%s\n----\n%s", enc, reEnc)
			}
		})
	}
}

func TestEncodeFromMapStringAny(t *testing.T) {
	// map[string]any keys are sorted alphabetically for determinism.
	in := map[string]any{"b": 2, "a": 1}
	got := mustEncode(t, in, nil)
	want := "a: 1\nb: 2"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEncodeFromStruct(t *testing.T) {
	type Hike struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Distance float64 `json:"distance"`
	}
	in := Hike{ID: 1, Name: "Blue Lake", Distance: 7.5}
	got := mustEncode(t, in, nil)
	want := "id: 1\nname: Blue Lake\ndistance: 7.5"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDecodeForbiddenLeadingZero(t *testing.T) {
	// "05" must decode as a string, not a number.
	in := "v: 05"
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	got, _ := m.Get("v")
	if got != "05" {
		t.Fatalf("got %#v want \"05\"", got)
	}
}

func TestDecodeStrictRejectsMultiplePrimitivesAtRoot(t *testing.T) {
	// §5: two or more non-empty depth-0 lines that are neither headers
	// nor key-value lines are invalid in strict mode.
	_, err := Decode("hello\nworld", nil)
	if err == nil {
		t.Fatal("expected error for multi-primitive root")
	}
}

func TestQuotingHyphenLeading(t *testing.T) {
	// Strings starting with '-' must be quoted.
	in := om("v", "-foo")
	got := mustEncode(t, in, nil)
	want := `v: "-foo"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDecodeBareKeyEmptyObject(t *testing.T) {
	// "x:" at depth 0 with no following indented line decodes as empty obj.
	in := "x:"
	v := mustDecode(t, in, nil)
	m := v.(*OrderedMap)
	x, _ := m.Get("x")
	if em, ok := x.(*OrderedMap); !ok || em.Len() != 0 {
		t.Fatalf("expected empty *OrderedMap, got %#v", x)
	}
}

func TestUnicode(t *testing.T) {
	in := om("name", "Adélie 🐧")
	got := mustEncode(t, in, nil)
	want := "name: Adélie 🐧"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	v := mustDecode(t, want, nil)
	m := v.(*OrderedMap)
	if n, _ := m.Get("name"); n != "Adélie 🐧" {
		t.Fatalf("got %#v", n)
	}
}

func TestEscapeRoundTrip(t *testing.T) {
	in := om("v", "line1\nline2\ttabbed")
	encoded := mustEncode(t, in, nil)
	decoded := mustDecode(t, encoded, nil)
	m := decoded.(*OrderedMap)
	got, _ := m.Get("v")
	if got != "line1\nline2\ttabbed" {
		t.Fatalf("escape round-trip: got %#v", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
