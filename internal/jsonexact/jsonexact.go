package jsonexact

import (
	"bytes"
	"encoding/json"
)

func Unmarshal[T any](d []byte, v *T) error {
	dec := json.NewDecoder(bytes.NewReader(d))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
