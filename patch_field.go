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

// IsSet 用来判断字段是否在请求中显式出现过。
func (f Field[T]) IsSet() bool {
	return f.Set
}

// HasValue 用来判断字段是否显式传值且不是 null。
func (f Field[T]) HasValue() (bool, T) {
	if f.Set && !f.Null {
		return true, f.Value
	}
	var zero T
	return false, zero
}

// UnmarshalJSON 用来支持 PATCH 场景下的三态字段解析。
func (f *Field[T]) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	f.Set = true
	if bytes.Equal(trimmed, []byte("null")) {
		f.Null = true
		var zero T
		f.Value = zero
		return nil
	}

	f.Null = false
	return json.Unmarshal(trimmed, &f.Value)
}
