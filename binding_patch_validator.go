package wd

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	patchValidationInterfaceType = reflect.TypeOf((*any)(nil)).Elem()
	patchValidationTimeType      = reflect.TypeOf(time.Time{})
)

// patchFieldStructValidator 用来让 gin 在校验前把 Field[T] 展开成真实值。
type patchFieldStructValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ binding.StructValidator = (*patchFieldStructValidator)(nil)

func (v *patchFieldStructValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.IsNil() {
			return nil
		}
		if value.Elem().Kind() != reflect.Struct {
			return v.ValidateStruct(value.Elem().Interface())
		}
		return v.validateStructValue(value.Elem())
	case reflect.Struct:
		return v.validateStructValue(value)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(binding.SliceValidationError, 0)
		for i := range count {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

func (v *patchFieldStructValidator) Engine() any {
	v.lazyinit()
	return v.validate
}

func (v *patchFieldStructValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

func (v *patchFieldStructValidator) validateStructValue(value reflect.Value) error {
	v.lazyinit()
	normalized := buildValidationStructValue(value)
	return v.validate.Struct(normalized.Interface())
}

func buildValidationStructValue(value reflect.Value) reflect.Value {
	value = dereferenceValidationValue(value)
	valueType := value.Type()

	fields := make([]reflect.StructField, 0, value.NumField())
	values := make([]reflect.Value, 0, value.NumField())

	for i := range value.NumField() {
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		fieldValue := value.Field(i)
		bindingTag := structField.Tag.Get("binding")
		jsonTag := structField.Tag.Get(TagJSON)

		if marker, ok := patchFieldValidationFromValue(fieldValue); ok {
			bindingTag = patchFieldBindingTag(bindingTag, marker)
		}

		fields = append(fields, reflect.StructField{
			Name:      structField.Name,
			Type:      patchValidationInterfaceType,
			Tag:       buildValidationStructTag(jsonTag, bindingTag),
			Anonymous: false,
		})
		values = append(values, patchValidationInterfaceValue(normalizeValidationValue(fieldValue)))
	}

	structType := reflect.StructOf(fields)
	structValue := reflect.New(structType).Elem()
	for i, fieldValue := range values {
		structValue.Field(i).Set(fieldValue)
	}

	return structValue
}

func normalizeValidationValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	if marker, ok := patchFieldValidationFromValue(value); ok {
		if !marker.patchFieldValidationSet() || marker.patchFieldValidationNull() {
			return nil
		}
		return normalizeValidationInterface(marker.patchFieldValidationValue())
	}

	value = dereferenceValidationValue(value)
	if !value.IsValid() {
		return nil
	}

	switch value.Kind() {
	case reflect.Struct:
		if value.Type().ConvertibleTo(patchValidationTimeType) {
			return value.Interface()
		}
		return buildValidationStructValue(value).Interface()
	case reflect.Slice:
		sliceValue := reflect.MakeSlice(reflect.SliceOf(patchValidationInterfaceType), value.Len(), value.Len())
		for i := range value.Len() {
			sliceValue.Index(i).Set(patchValidationInterfaceValue(normalizeValidationValue(value.Index(i))))
		}
		return sliceValue.Interface()
	case reflect.Array:
		arrayType := reflect.ArrayOf(value.Len(), patchValidationInterfaceType)
		arrayValue := reflect.New(arrayType).Elem()
		for i := range value.Len() {
			arrayValue.Index(i).Set(patchValidationInterfaceValue(normalizeValidationValue(value.Index(i))))
		}
		return arrayValue.Interface()
	case reflect.Map:
		mapType := reflect.MapOf(value.Type().Key(), patchValidationInterfaceType)
		if value.IsNil() {
			return reflect.Zero(mapType).Interface()
		}
		mapValue := reflect.MakeMapWithSize(mapType, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			mapValue.SetMapIndex(iter.Key(), patchValidationInterfaceValue(normalizeValidationValue(iter.Value())))
		}
		return mapValue.Interface()
	default:
		return value.Interface()
	}
}

func normalizeValidationInterface(value any) any {
	if value == nil {
		return nil
	}
	return normalizeValidationValue(reflect.ValueOf(value))
}

func patchValidationInterfaceValue(value any) reflect.Value {
	if value == nil {
		return reflect.Zero(patchValidationInterfaceType)
	}
	return reflect.ValueOf(value)
}

func patchFieldValidationFromValue(value reflect.Value) (patchFieldValidationMarker, bool) {
	if !value.IsValid() {
		return nil, false
	}
	value = dereferenceValidationValue(value)
	if !value.IsValid() || !value.CanInterface() {
		return nil, false
	}
	marker, ok := value.Interface().(patchFieldValidationMarker)
	return marker, ok
}

func dereferenceValidationValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func patchFieldBindingTag(bindingTag string, marker patchFieldValidationMarker) string {
	if marker.patchFieldValidationSet() {
		return bindingTag
	}
	if !shouldInjectPatchFieldOmitEmpty(bindingTag) {
		return bindingTag
	}
	return "omitempty," + bindingTag
}

func shouldInjectPatchFieldOmitEmpty(bindingTag string) bool {
	if bindingTag == "" || bindingTag == "-" {
		return false
	}

	for _, item := range strings.Split(bindingTag, ",") {
		name := item
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		switch name {
		case "omitempty", "omitnil", "omitzero":
			return false
		}
		if name == "required" || strings.HasPrefix(name, "required_") || strings.HasPrefix(name, "excluded_") || name == "skip_unless" {
			return false
		}
	}

	return true
}

func buildValidationStructTag(jsonTag, bindingTag string) reflect.StructTag {
	parts := make([]string, 0, 2)
	if jsonTag != "" {
		parts = append(parts, fmt.Sprintf(`json:"%s"`, escapeValidationTagValue(jsonTag)))
	}
	if bindingTag != "" {
		parts = append(parts, fmt.Sprintf(`binding:"%s"`, escapeValidationTagValue(bindingTag)))
	}
	return reflect.StructTag(strings.Join(parts, " "))
}

func escapeValidationTagValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}
