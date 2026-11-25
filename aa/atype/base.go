package atype

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"strconv"
)

// MarshalUint8s converts a uint8 slice to a JSON array of numbers.
// encoding/json/v2 represents []byte as Base64; this keeps a numeric array.
func MarshalUint8s(v []uint8) (jsontext.Value, bool) {
	if len(v) == 0 {
		return nil, true
	}

	// Preallocate enough space
	result := make([]byte, 0, len(v)*4+2)
	result = append(result, '[')

	for i, x := range v {
		if i > 0 {
			result = append(result, ',')
		}
		result = strconv.AppendUint(result, uint64(x), 10)
	}

	result = append(result, ']')
	return result, true
}

func UnmarshalUint8s(x jsontext.Value) ([]uint8, error) {
	s := bytes.TrimSpace(x)
	if len(s) == 0 || string(s) == "null" {
		return nil, nil
	}
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, errors.New("invalid uint8 json: " + string(s))
	}

	body := s[1 : len(s)-1]
	v := make([]uint8, 0, 8)
	for i := 0; i < len(body); {
		for i < len(body) && (body[i] == ' ' || body[i] == ',' || body[i] == '\t' || body[i] == '\r' || body[i] == '\n') {
			i++
		}
		if i >= len(body) {
			break
		}
		start := i
		for i < len(body) && body[i] >= '0' && body[i] <= '9' {
			i++
		}
		if start == i {
			return nil, errors.New("invalid uint8 json: " + string(s))
		}
		n, err := strconv.ParseUint(string(body[start:i]), 10, 8)
		if err != nil {
			return nil, err
		}
		v = append(v, uint8(n))
	}
	return v, nil
}
