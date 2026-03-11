package wd

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"gorm.io/gen/field"
)

// AppendPatchAuto 用来根据 oldValue 自动判断字段是否可为空。
// oldValue 传模型字段旧值本身即可：非指针表示不可为空，指针表示可为空。
// setValue 可以是 func(T) field.AssignExpr，也可以是更宽的单参数函数，例如 func(driver.Valuer) field.AssignExpr。
// 对于可空字段，需要额外传入 setNull。
func AppendPatchAuto[T comparable](updates []field.AssignExpr, patch Field[T], oldValue any, setValue any, setNull ...func() field.AssignExpr) []field.AssignExpr {
	oldInfo := parsePatchOldValue[T](oldValue)
	if oldInfo.nullable {
		if len(setNull) == 0 || setNull[0] == nil {
			panic("可空字段必须提供 setNull")
		}
		if !patch.IsSet() {
			return updates
		}
		if oldInfo.isNull && patch.Null {
			return updates
		}
		if !oldInfo.isNull && !patch.Null && oldInfo.value == patch.Value {
			return updates
		}
		if ok, value := patch.HasValue(); ok {
			return append(updates, callPatchSetValue(setValue, value))
		}
		return append(updates, setNull[0]())
	}

	if ok, value := patch.HasValue(); ok && value != oldInfo.value {
		return append(updates, callPatchSetValue(setValue, value))
	}
	return updates
}

// AppendPatchValue 用来处理不可为空的 PATCH 字段。
// 只有请求显式传值且与旧值不同，才会追加更新表达式。
func AppendPatchValue[T comparable](updates []field.AssignExpr, patch Field[T], oldValue T, setValue func(T) field.AssignExpr) []field.AssignExpr {
	return AppendPatchAuto(updates, patch, oldValue, setValue)
}

// AppendPatchNullable 用来处理允许为 null 的 PATCH 字段。
// 只有请求显式传入且与旧值语义不同，才会追加更新表达式。
func AppendPatchNullable[T comparable](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue func(T) field.AssignExpr, setNull func() field.AssignExpr) []field.AssignExpr {
	return AppendPatchAuto(updates, patch, oldValue, setValue, setNull)
}

// AppendPatchNullableAuto 用来自动适配不同 Value 方法签名的可空 PATCH 字段。
// setValue 可以是 func(T) field.AssignExpr，也可以是 func(interface{...}) field.AssignExpr。
func AppendPatchNullableAuto[T comparable](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue any, setNull func() field.AssignExpr) []field.AssignExpr {
	return AppendPatchAuto(updates, patch, oldValue, setValue, setNull)
}

// AppendPatchNullableValuer 用来处理允许为 null 且底层使用 driver.Valuer 的 PATCH 字段。
// 只有请求显式传入且与旧值语义不同，才会追加更新表达式。
func AppendPatchNullableValuer[T interface {
	comparable
	driver.Valuer
}](updates []field.AssignExpr, patch Field[T], oldValue *T, setValue func(driver.Valuer) field.AssignExpr, setNull func() field.AssignExpr) []field.AssignExpr {
	return AppendPatchAuto(updates, patch, oldValue, setValue, setNull)
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

type patchOldValueInfo[T comparable] struct {
	value    T
	nullable bool
	isNull   bool
}

func parsePatchOldValue[T comparable](oldValue any) patchOldValueInfo[T] {
	value := reflect.ValueOf(oldValue)
	if !value.IsValid() {
		panic("oldValue 不能为空，请传入模型字段旧值")
	}

	if value.Kind() == reflect.Ptr {
		var info patchOldValueInfo[T]
		info.nullable = true
		if value.IsNil() {
			info.isNull = true
			return info
		}
		info.value = castPatchOldValue[T](value.Elem())
		return info
	}

	return patchOldValueInfo[T]{
		value: castPatchOldValue[T](value),
	}
}

func castPatchOldValue[T any](value reflect.Value) T {
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	if !value.Type().AssignableTo(targetType) {
		if !value.Type().ConvertibleTo(targetType) {
			panic(fmt.Sprintf("oldValue 类型不匹配: need=%s got=%s", targetType, value.Type()))
		}
		value = value.Convert(targetType)
	}
	return value.Interface().(T)
}
