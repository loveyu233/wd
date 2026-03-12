package wd

import (
	"fmt"
	"reflect"

	"gorm.io/gen/field"
)

// AppendPatchUpdate 用来根据 oldValue 自动判断字段是否可为空。
// oldValue 传模型字段旧值本身即可：非指针表示不可为空，指针表示可为空。
// target 支持两种传法：
// 1. 直接传字段对象，例如 query.User.Email，helper 会自动调用 Value/Null
// 2. 传 Value 方法；这种模式下如果字段可空，仍需要额外传入 setNull
func AppendPatchUpdate[T comparable](patch Field[T], oldValue any, target any, setNull ...func() field.AssignExpr) (field.AssignExpr, bool) {
	oldInfo := parsePatchOldValue[T](oldValue)
	if oldInfo.nullable {
		if len(setNull) == 0 || setNull[0] == nil {
			if hasPatchColumnMethods(target) {
				setNull = []func() field.AssignExpr{func() field.AssignExpr {
					return callPatchColumnNull(target)
				}}
			}
		}
		if len(setNull) == 0 || setNull[0] == nil {
			panic("可空字段必须提供 setNull")
		}
		if !patch.IsSet() {
			return nil, false
		}
		if oldInfo.isNull && patch.Null {
			return nil, false
		}
		if !oldInfo.isNull && !patch.Null && oldInfo.value == patch.Value {
			return nil, false
		}
		if ok, value := patch.HasValue(); ok {
			return callPatchTargetValue(target, value), true
		}
		return setNull[0](), true
	}

	if ok, value := patch.HasValue(); ok && value != oldInfo.value {
		return callPatchTargetValue(target, value), true
	}
	return nil, false
}

func callPatchTargetValue[T any](target any, value T) field.AssignExpr {
	if reflect.ValueOf(target).Kind() == reflect.Func {
		return callPatchSetValue(target, value)
	}
	return callPatchColumnValue(target, value)
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

func callPatchColumnValue[T any](column any, value T) field.AssignExpr {
	columnValue := reflect.ValueOf(column)
	if !columnValue.IsValid() {
		panic("column 不能为空")
	}
	method := columnValue.MethodByName("Value")
	if !method.IsValid() {
		panic("column 必须包含 Value 方法")
	}
	return callPatchMethod(method, value)
}

func callPatchColumnNull(column any) field.AssignExpr {
	columnValue := reflect.ValueOf(column)
	if !columnValue.IsValid() {
		panic("column 不能为空")
	}
	method := columnValue.MethodByName("Null")
	if !method.IsValid() {
		panic("column 必须包含 Null 方法")
	}
	if method.Type().NumIn() != 0 || method.Type().NumOut() != 1 {
		panic("column.Null 方法签名不合法")
	}
	result := method.Call(nil)
	assignExpr, ok := result[0].Interface().(field.AssignExpr)
	if !ok {
		panic("column.Null 返回值必须实现 field.AssignExpr")
	}
	return assignExpr
}

func hasPatchColumnMethods(column any) bool {
	columnValue := reflect.ValueOf(column)
	if !columnValue.IsValid() || columnValue.Kind() == reflect.Func {
		return false
	}
	return columnValue.MethodByName("Value").IsValid() && columnValue.MethodByName("Null").IsValid()
}

func callPatchMethod[T any](method reflect.Value, value T) field.AssignExpr {
	methodType := method.Type()
	if methodType.NumIn() != 1 || methodType.NumOut() != 1 {
		panic("方法签名必须是单入参单返回值")
	}

	arg := reflect.ValueOf(value)
	paramType := methodType.In(0)
	if !arg.Type().AssignableTo(paramType) {
		if !arg.Type().ConvertibleTo(paramType) {
			panic(fmt.Sprintf("方法参数类型不匹配: need=%s got=%s", paramType, arg.Type()))
		}
		arg = arg.Convert(paramType)
	}

	result := method.Call([]reflect.Value{arg})
	assignExpr, ok := result[0].Interface().(field.AssignExpr)
	if !ok {
		panic("方法返回值必须实现 field.AssignExpr")
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
