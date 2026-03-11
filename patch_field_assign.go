package wd

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"gorm.io/gen/field"
)

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

// AppendPatchNullableAuto 用来自动适配不同 Value 方法签名的可空 PATCH 字段。
// setValue 可以是 func(T) field.AssignExpr，也可以是 func(interface{...}) field.AssignExpr。
func AppendPatchNullableAuto[T comparable](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue any, setNull func() field.AssignExpr) []field.AssignExpr {
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
		return append(updates, callPatchSetValue(setValue, value))
	}
	return append(updates, setNull())
}

// AppendPatchNullableValuer 用来处理允许为 null 且底层使用 driver.Valuer 的 PATCH 字段。
// 只有请求显式传入且与旧值语义不同，才会追加更新表达式。
func AppendPatchNullableValuer[T interface {
	comparable
	driver.Valuer
}](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue func(driver.Valuer) field.AssignExpr, setNull func() field.AssignExpr) []field.AssignExpr {
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

func callPatchSetValue[T any](setValue any, value T) field.AssignExpr {
	setter := reflect.ValueOf(setValue)
	if !setter.IsValid() || setter.Kind() != reflect.Func {
		panic("setValue 必须是函数")
	}

	setterType := setter.Type()
	if setterType.NumIn() != 1 || setterType.NumOut() != 1 {
		panic("setValue 必须是单入参单返回值函数")
	}

	arg := reflect.ValueOf(value)
	paramType := setterType.In(0)
	if !arg.Type().AssignableTo(paramType) {
		if !arg.Type().ConvertibleTo(paramType) {
			panic(fmt.Sprintf("setValue 参数类型不匹配: need=%s got=%s", paramType, arg.Type()))
		}
		arg = arg.Convert(paramType)
	}

	result := setter.Call([]reflect.Value{arg})
	assignExpr, ok := result[0].Interface().(field.AssignExpr)
	if !ok {
		panic("setValue 返回值必须实现 field.AssignExpr")
	}
	return assignExpr
}
