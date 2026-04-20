package wd

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type PatchEqualFunc[T any] func(oldValue, newValue T) bool

type patchValueTarget[T any] interface {
	Value(T) field.AssignExpr
}

type patchNullTarget interface {
	Null() field.AssignExpr
}

type patchDriverValuerTarget interface {
	Value(driver.Valuer) field.AssignExpr
}

type patchSerializerValueTarget interface {
	Value(schema.SerializerValuerInterface) field.AssignExpr
}

type patchResolvedTarget[T any] struct {
	setValue func(T) (field.AssignExpr, error)
	setNull  func() field.AssignExpr
}

func PatchUpdateSimple[T any](patch Field[T], oldValue any, target any, setNull ...func() field.AssignExpr) (field.AssignExpr, error) {
	ae, updated, err := patchUpdateWithEqual(patch, oldValue, target, defaultPatchEqual[T], firstPatchNullSetter(setNull...))
	if err != nil || !updated {
		return nil, err
	}
	return ae, nil
}

// PatchUpdate 判断新旧两个字段是否相同，如果不相同则创建修改，相同则直接返回。
func PatchUpdate[T any](patch Field[T], oldValue any, target any, setNull ...func() field.AssignExpr) (ae field.AssignExpr, isUpdate bool, err error) {
	return patchUpdateWithEqual(patch, oldValue, target, defaultPatchEqual[T], firstPatchNullSetter(setNull...))
}

// PatchUpdateSimpleBy 用来自定义两个值的比较逻辑，适合 decimal、JSON 等需要业务等价判断的类型。
func PatchUpdateSimpleBy[T any](patch Field[T], oldValue any, target any, equal PatchEqualFunc[T], setNull ...func() field.AssignExpr) (field.AssignExpr, error) {
	ae, updated, err := patchUpdateWithEqual(patch, oldValue, target, equal, firstPatchNullSetter(setNull...))
	if err != nil || !updated {
		return nil, err
	}
	return ae, nil
}

// PatchUpdateBy 用来自定义两个值的比较逻辑，适合 decimal、JSON 等需要业务等价判断的类型。
func PatchUpdateBy[T any](patch Field[T], oldValue any, target any, equal PatchEqualFunc[T], setNull ...func() field.AssignExpr) (ae field.AssignExpr, isUpdate bool, err error) {
	return patchUpdateWithEqual(patch, oldValue, target, equal, firstPatchNullSetter(setNull...))
}

func patchUpdateWithEqual[T any](patch Field[T], oldValue any, target any, equal PatchEqualFunc[T], setNull func() field.AssignExpr) (field.AssignExpr, bool, error) {
	if equal == nil {
		equal = defaultPatchEqual[T]
	}

	oldInfo, err := parsePatchOldValue[T](oldValue)
	if err != nil {
		return nil, false, err
	}

	resolvedTarget, err := resolvePatchTarget[T](target)
	if err != nil {
		return nil, false, err
	}
	if setNull != nil {
		resolvedTarget.setNull = setNull
	}

	return patchApplyUpdate(
		patch.Set,
		patch.Null,
		func() (any, bool) {
			ok, value := patch.HasValue()
			if !ok {
				return nil, false
			}
			return value, true
		},
		patchOldValueState{
			known:    true,
			value:    oldInfo.value,
			nullable: oldInfo.nullable,
			isNull:   oldInfo.isNull,
		},
		func(oldValue, newValue any) bool {
			return equal(oldValue.(T), newValue.(T))
		},
		func(value any) (field.AssignExpr, error) {
			return resolvedTarget.setValue(value.(T))
		},
		func() (field.AssignExpr, error) {
			if resolvedTarget.setNull == nil {
				return nil, errors.New("可空字段必须提供 setNull 或目标字段支持 Null()")
			}
			return resolvedTarget.setNull(), nil
		},
		"可空字段必须提供 setNull 或目标字段支持 Null()",
	)
}

func resolvePatchTarget[T any](target any) (patchResolvedTarget[T], error) {
	if target == nil {
		return patchResolvedTarget[T]{}, errors.New("target 不能为空")
	}

	resolved := patchResolvedTarget[T]{
		setNull: patchNullSetterFromTarget(target),
	}

	switch typedTarget := target.(type) {
	case func(T) field.AssignExpr:
		resolved.setValue = func(value T) (field.AssignExpr, error) {
			return typedTarget(value), nil
		}
		return resolved, nil
	case patchValueTarget[T]:
		resolved.setValue = func(value T) (field.AssignExpr, error) {
			return typedTarget.Value(value), nil
		}
		return resolved, nil
	case patchDriverValuerTarget:
		resolved.setValue = func(value T) (field.AssignExpr, error) {
			valuer, ok := patchAsDriverValuer(value)
			if !ok {
				return nil, fmt.Errorf("target.Value 需要 driver.Valuer，但当前类型 %s 不支持", patchTypeName[T]())
			}
			return typedTarget.Value(valuer), nil
		}
		return resolved, nil
	case patchSerializerValueTarget:
		resolved.setValue = func(value T) (field.AssignExpr, error) {
			serializerValue, ok := patchAsSerializerValue(value)
			if !ok {
				return nil, fmt.Errorf("target.Value 需要 schema.SerializerValuerInterface，但当前类型 %s 不支持", patchTypeName[T]())
			}
			return typedTarget.Value(serializerValue), nil
		}
		return resolved, nil
	default:
		return patchResolvedTarget[T]{}, fmt.Errorf("target 类型不支持: %T", target)
	}
}

func patchNullSetterFromTarget(target any) func() field.AssignExpr {
	nullableTarget, ok := target.(patchNullTarget)
	if !ok {
		return nil
	}
	return nullableTarget.Null
}

func patchAsDriverValuer[T any](value T) (driver.Valuer, bool) {
	if valuer, ok := any(value).(driver.Valuer); ok {
		return valuer, true
	}
	if valuer, ok := any(&value).(driver.Valuer); ok {
		return valuer, true
	}
	return nil, false
}

func patchAsSerializerValue[T any](value T) (schema.SerializerValuerInterface, bool) {
	if serializerValue, ok := any(value).(schema.SerializerValuerInterface); ok {
		return serializerValue, true
	}
	if serializerValue, ok := any(&value).(schema.SerializerValuerInterface); ok {
		return serializerValue, true
	}
	return nil, false
}

type patchOldValueInfo[T any] struct {
	value    T
	nullable bool
	isNull   bool
}

func parsePatchOldValue[T any](oldValue any) (patchOldValueInfo[T], error) {
	if oldValue == nil {
		return patchOldValueInfo[T]{
			nullable: true,
			isNull:   true,
		}, nil
	}

	switch value := oldValue.(type) {
	case T:
		return patchOldValueInfo[T]{value: value}, nil
	case *T:
		info := patchOldValueInfo[T]{nullable: true}
		if value == nil {
			info.isNull = true
			return info, nil
		}
		info.value = *value
		return info, nil
	default:
		return patchOldValueInfo[T]{}, fmt.Errorf("oldValue 类型不匹配: need=%s 或 *%s got=%T", patchTypeName[T](), patchTypeName[T](), oldValue)
	}
}

func defaultPatchEqual[T any](oldValue, newValue T) bool {
	return reflect.DeepEqual(oldValue, newValue)
}

func firstPatchNullSetter(setNull ...func() field.AssignExpr) func() field.AssignExpr {
	if len(setNull) == 0 || setNull[0] == nil {
		return nil
	}
	return setNull[0]
}

func patchTypeName[T any]() string {
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	return targetType.String()
}
