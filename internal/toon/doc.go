// Package toon implements TOON (Token-Oriented Object Notation), a
// compact, indentation-based encoding of the JSON data model designed
// for LLM prompts.
//
// # Format overview
//
// TOON expresses the same primitive/object/array model as JSON. Its
// advantages come from declaring array shapes once (length and optional
// field list) and using indentation in place of braces:
//
//	context:
//	  task: Our favorite hikes
//	  location: Boulder
//	friends[3]: ana,luis,sam
//	hikes[2]{id,name,distanceKm}:
//	  1,Blue Lake,7.5
//	  2,Ridge Overlook,9.2
//
// The reference specification (v3.3) lives at
// https://github.com/toon-format/spec/blob/main/SPEC.md. This package
// targets that document.
//
// # API
//
// Encode serializes a Go value into a TOON document; Decode parses a
// TOON document into Go values. Both round-trip losslessly under the
// JSON data model.
//
//	out, err := toon.Encode(value, nil)
//	v, err  := toon.Decode(out, nil)
//
// Encode accepts:
//   - nil, bool, all integer and floating types, string;
//   - []any, or any slice/array via reflection;
//   - map[string]any (keys sorted alphabetically for deterministic
//     output), *OrderedMap (insertion-order preserved), or a struct
//     (fields in source order, honoring `json` tags).
//
// Decode returns:
//   - nil, bool, float64, string for primitives;
//   - []any for arrays;
//   - *OrderedMap for objects, so key order is preserved per spec §2.
//
// # Design notes
//
// TOON requires that object key order be preserved on both encode and
// decode. Go's built-in map does not guarantee this, so OrderedMap is
// the canonical object type for round-trip workflows. Callers that
// don't care about order can pass map[string]any to Encode and call
// (*OrderedMap).Map() on decoded values.
//
// Numbers follow the spec's canonical decimal form on encode: integer-
// valued floats lose their trailing zero (1.0 → "1"), -0 normalizes to
// 0, and values outside [1e-6, 1e21) use exponent form. On decode all
// numbers come back as float64, matching encoding/json semantics.
//
// Tabular detection is automatic: any array whose elements are uniform
// non-empty objects with primitive-only values is emitted using the
// `[N]{fields}:` header form. Other arrays use the inline (primitives)
// or expanded list form as appropriate.
//
// # Limitations
//
// The optional key folding (encoder) and path expansion (decoder)
// features from §13.4 are not implemented. The encoder defaults to LF
// line endings, 2-space indentation, and comma as the document
// delimiter; these are configurable via EncodeOptions.
package toon
