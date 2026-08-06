// Package canonicalize implements the exact JSON canonicalization contract used
// by two of the three G4 fingerprint algorithms (the source-profiler fingerprint
// and the governed fingerprint). It reproduces Python's:
//
//	json.dumps(value, sort_keys=True, ensure_ascii=True, separators=(",", ":"))
//
// byte-for-byte. The composite installation digest is binary-framed and does NOT
// use this canonicalization; it is implemented separately in the migration
// package's CompositeDigest function.
//
// The frozen projections hash only ints, strings, bools, arrays and objects.
// Floats are rejected (no string-encoding fallback); if a frozen projection ever
// introduces a float, that is a G4 stop, not a serialization workaround.
package canonicalize

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"unicode/utf16"
)

// ErrFloatUnsupported is returned when a value to be canonicalized contains a
// float. No frozen projection contains one; introducing one is a G4 stop.
var ErrFloatUnsupported = errors.New("canonicalize: float values are not supported by the frozen projections")

// Marshal returns the canonical bytes for v. It mirrors Python json.dumps with
// sort_keys=True, ensure_ascii=True, separators=(",",":"): compact (no
// whitespace), keys sorted by Unicode code point at every depth, all non-ASCII
// escaped as \uXXXX (with surrogate pairs for astral planes), HTML chars (<>&)
// NOT escaped, and no trailing newline.
func Marshal(v any) ([]byte, error) {
	var buf []byte
	if err := appendValue(&buf, v); err != nil {
		return nil, err
	}
	return buf, nil
}

func appendValue(buf *[]byte, v any) error {
	switch t := v.(type) {
	case nil:
		*buf = append(*buf, 'n', 'u', 'l', 'l')
		return nil
	case bool:
		if t {
			*buf = append(*buf, 't', 'r', 'u', 'e')
		} else {
			*buf = append(*buf, 'f', 'a', 'l', 's', 'e')
		}
		return nil
	case int:
		*buf = strconv.AppendInt(*buf, int64(t), 10)
		return nil
	case int32:
		*buf = strconv.AppendInt(*buf, int64(t), 10)
		return nil
	case int64:
		*buf = strconv.AppendInt(*buf, t, 10)
		return nil
	case uint:
		*buf = strconv.AppendUint(*buf, uint64(t), 10)
		return nil
	case uint64:
		*buf = strconv.AppendUint(*buf, t, 10)
		return nil
	case float32:
		return ErrFloatUnsupported
	case float64:
		return ErrFloatUnsupported
	case json.Number:
		return appendJSONNumber(buf, t)
	case string:
		appendPythonString(buf, t)
		return nil
	case []byte:
		// Treat raw bytes as a JSON string of the bytes. The frozen projections
		// do not produce []byte values; if one appears, encode it as a UTF-8
		// string the way Python would decode it.
		appendPythonString(buf, string(t))
		return nil
	case map[string]any:
		return appendObject(buf, t)
	case []any:
		return appendArray(buf, t)
	case []string:
		arr := make([]any, len(t))
		for i, s := range t {
			arr[i] = s
		}
		return appendArray(buf, arr)
	case []int:
		arr := make([]any, len(t))
		for i, n := range t {
			arr[i] = n
		}
		return appendArray(buf, arr)
	case []int64:
		arr := make([]any, len(t))
		for i, n := range t {
			arr[i] = n
		}
		return appendArray(buf, arr)
	case []map[string]any:
		arr := make([]any, len(t))
		for i, m := range t {
			arr[i] = m
		}
		return appendArray(buf, arr)
	default:
		// Fall back to json.Marshal for unknown struct/map types, then
		// re-canonicalize through a decoded generic value. This keeps the
		// public API ergonomic for callers passing structs.
		raw, err := json.Marshal(t)
		if err != nil {
			return fmt.Errorf("canonicalize: unsupported type %T: %w", v, err)
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var generic any
		if err := dec.Decode(&generic); err != nil {
			return fmt.Errorf("canonicalize: decode fallback: %w", err)
		}
		return appendValue(buf, generic)
	}
}

// appendJSONNumber handles json.Number from UseNumber decoding. It must match
// Python's integer formatting (the only numeric type in scope). A number with a
// decimal point or exponent would be a float and is rejected.
func appendJSONNumber(buf *[]byte, n json.Number) error {
	s := string(n)
	if containsFloatMarker(s) {
		return fmt.Errorf("%w: %s", ErrFloatUnsupported, s)
	}
	*buf = append(*buf, s...)
	return nil
}

func containsFloatMarker(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' || c == 'e' || c == 'E' {
			return true
		}
	}
	return false
}

func appendArray(buf *[]byte, arr []any) error {
	*buf = append(*buf, '[')
	for i, e := range arr {
		if i > 0 {
			*buf = append(*buf, ',')
		}
		if err := appendValue(buf, e); err != nil {
			return err
		}
	}
	*buf = append(*buf, ']')
	return nil
}

func appendObject(buf *[]byte, m map[string]any) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Python sort_keys sorts by Unicode code point. Go's string < is a byte
	// comparison; for valid UTF-8 this matches code-point order.
	sort.Strings(keys)

	*buf = append(*buf, '{')
	for i, k := range keys {
		if i > 0 {
			*buf = append(*buf, ',')
		}
		appendPythonString(buf, k)
		*buf = append(*buf, ':')
		if err := appendValue(buf, m[k]); err != nil {
			return err
		}
	}
	*buf = append(*buf, '}')
	return nil
}

// appendPythonString writes a JSON string matching Python's ensure_ascii=True
// escaping: every rune > 0x7F becomes \uXXXX, with astral planes (>0xFFFF)
// encoded as UTF-16 surrogate pairs. Control characters use Python's short
// forms (\n,\t,\r,\b,\f) and others use \u00XX. Forward slash is NOT escaped
// (Python does not escape it). HTML chars (<>&) are NOT escaped (Python default).
func appendPythonString(buf *[]byte, s string) {
	*buf = append(*buf, '"')
	for _, r := range s {
		switch r {
		case '"':
			*buf = append(*buf, '\\', '"')
		case '\\':
			*buf = append(*buf, '\\', '\\')
		case '\n':
			*buf = append(*buf, '\\', 'n')
		case '\r':
			*buf = append(*buf, '\\', 'r')
		case '\t':
			*buf = append(*buf, '\\', 't')
		case '\b':
			*buf = append(*buf, '\\', 'b')
		case '\f':
			*buf = append(*buf, '\\', 'f')
		default:
			if r < 0x20 {
				// Other control chars: \u00XX
				*buf = append(*buf, '\\', 'u', '0', '0', hexDigit(byte(r>>4)), hexDigit(byte(r&0xF)))
			} else if r < 0x7F {
				*buf = append(*buf, byte(r))
			} else if r <= 0xFFFF {
				// BMP non-ASCII: \uXXXX
				appendU4(buf, uint16(r))
			} else {
				// Astral plane: UTF-16 surrogate pair.
				r1, r2 := utf16.EncodeRune(r)
				appendU4(buf, uint16(r1))
				appendU4(buf, uint16(r2))
			}
		}
	}
	*buf = append(*buf, '"')
}

func appendU4(buf *[]byte, u uint16) {
	*buf = append(*buf, '\\', 'u')
	*buf = append(*buf, hexDigit(byte(u>>12)), hexDigit(byte(u>>8&0xF)), hexDigit(byte(u>>4&0xF)), hexDigit(byte(u&0xF)))
}

func hexDigit(b byte) byte {
	if b < 10 {
		return '0' + b
	}
	return 'a' + (b - 10)
}
