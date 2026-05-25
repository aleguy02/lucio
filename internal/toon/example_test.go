package toon_test

import (
	"encoding/json"
	"fmt"

	"aleguy02/spotify-tui/internal/toon"
)

func ExampleEncode() {
	doc := toon.NewOrderedMap()
	doc.Set("id", 123)
	doc.Set("name", "Ada")
	doc.Set("active", true)

	out, _ := toon.Encode(doc, nil)
	fmt.Println(out)
	// Output:
	// id: 123
	// name: Ada
	// active: true
}

func ExampleEncode_tabular() {
	rows := []any{
		mustOM("sku", "A1", "qty", 2, "price", 9.99),
		mustOM("sku", "B2", "qty", 1, "price", 14.5),
	}
	doc := toon.NewOrderedMap()
	doc.Set("items", rows)

	out, _ := toon.Encode(doc, nil)
	fmt.Println(out)
	// Output:
	// items[2]{sku,qty,price}:
	//   A1,2,9.99
	//   B2,1,14.5
}

func ExampleEncode_pipeDelimiter() {
	doc := toon.NewOrderedMap()
	doc.Set("tags", []any{"red", "green", "blue"})

	out, _ := toon.Encode(doc, &toon.EncodeOptions{Delimiter: toon.DelimiterPipe})
	fmt.Println(out)
	// Output:
	// tags[3|]: red|green|blue
}

func ExampleDecode() {
	src := "user:\n  id: 1\n  name: Bob"
	v, _ := toon.Decode(src, nil)

	m := v.(*toon.OrderedMap)
	user, _ := m.Get("user")
	um := user.(*toon.OrderedMap)
	name, _ := um.Get("name")
	fmt.Println(name)
	// Output: Bob
}

// ExampleEncode_fromJSON encodes a value parsed from JSON into TOON.
// JSON parsing through encoding/json into map[string]any loses key
// order; the encoder sorts keys alphabetically for deterministic
// output in that case. Callers who need to preserve order should
// build an OrderedMap directly.
func ExampleEncode_fromJSON() {
	jsonSrc := `{"id":1,"name":"Ada","tags":["admin","ops"]}`

	var raw any
	_ = json.Unmarshal([]byte(jsonSrc), &raw)

	out, _ := toon.Encode(raw, nil)
	fmt.Println(out)
	// Output:
	// id: 1
	// name: Ada
	// tags[2]: admin,ops
}

// ExampleEncode_struct uses a Go struct so TOON output preserves the
// fields' declaration order (json tags rename fields).
func ExampleEncode_struct() {
	type Track struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Duration int    `json:"duration_ms"`
	}
	out, _ := toon.Encode(Track{ID: 7, Name: "Yesterday", Duration: 125000}, nil)
	fmt.Println(out)
	// Output:
	// id: 7
	// name: Yesterday
	// duration_ms: 125000
}

func mustOM(pairs ...any) *toon.OrderedMap {
	m := toon.NewOrderedMap()
	for i := 0; i < len(pairs); i += 2 {
		m.Set(pairs[i].(string), pairs[i+1])
	}
	return m
}
