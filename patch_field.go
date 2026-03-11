package wd

import (
	"bytes"
	"encoding/json"
)

// Field 用来区分字段是否传入、是否显式传入 null，以及最终的新值。
type Field[T any] struct {
	Set   bool
	Null  bool
	Value T
}

// UnmarshalJSON 用来支持 PATCH 场景下的三态字段解析。
func (f *Field[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	if bytes.Equal(bytes.ReplaceAll(bytes.TrimSpace(data), []byte("\""), nil), []byte("null")) {
		f.Null = true
		var zero T
		f.Value = zero
		return nil
	}

	f.Null = false
	return json.Unmarshal(data, &f.Value)
}
