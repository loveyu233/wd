package wd

import "gorm.io/gen/field"

// AppendPatchValue 用来处理不可为空的 PATCH 字段。
// 只有请求显式传值且与旧值不同，才会追加更新表达式。
func AppendPatchValue[T comparable](updates []field.AssignExpr, patch Field[T], oldValue T, setValue func(T) field.AssignExpr) []field.AssignExpr {
	if ok, value := patch.HasValue(); ok && value != oldValue {
		return append(updates, setValue(value))
	}
	return updates
}

// AppendPatchNullable 用来处理允许为 null 的 PATCH 字段。
// 只有请求显式传入且与旧值语义不同，才会追加更新表达式。
func AppendPatchNullable[T comparable](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue func(T) field.AssignExpr, setNull func() field.AssignExpr) []field.AssignExpr {
	if !patch.IsSet() {
		return updates
	}
	if oldValue == nil && patch.Null {
		return updates
	}
	if oldValue != nil && !patch.Null && *oldValue == patch.Value {
		return updates
	}
	if ok, value := patch.HasValue(); ok {
		return append(updates, setValue(value))
	}
	return append(updates, setNull())
}
