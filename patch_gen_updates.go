package wd

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gen/field"
)

const patchTag = "patch"

var patchAssignExprType = reflect.TypeOf((*field.AssignExpr)(nil)).Elem()

type patchUnknownOldValue struct{}

type patchGetFieldByName interface {
	GetFieldByName(string) (field.OrderExpr, bool)
}

type patchDynamicOldValueInfo struct {
	known    bool
	value    any
	nullable bool
	isNull   bool
}

type patchResolvedDynamicTarget struct {
	setValue func(any) (field.AssignExpr, error)
	setNull  func() (field.AssignExpr, error)
}

// BuildGenUpdates 用来把 PATCH 请求中的 Field[T] 一次性转换成 gorm/gen 的更新表达式。
// oldModel 允许传 nil，表示当前没有旧值可比较，此时只要请求显式传了字段就会生成更新表达式。
func BuildGenUpdates(req any, oldModel any, table any) ([]field.AssignExpr, error) {
	reqValue, err := patchBuildStructValue(req, "req")
	if err != nil {
		return nil, err
	}

	tableValue, err := patchBuildStructValue(table, "table")
	if err != nil {
		return nil, err
	}

	oldValue, hasOldModel, err := patchBuildOptionalStructValue(oldModel)
	if err != nil {
		return nil, err
	}

	updates := make([]field.AssignExpr, 0)
	if err := buildGenUpdatesFromStruct(&updates, reqValue, oldValue, hasOldModel, table, tableValue); err != nil {
		return nil, err
	}
	return updates, nil
}

func buildGenUpdatesFromStruct(
	updates *[]field.AssignExpr,
	reqValue reflect.Value,
	oldValue reflect.Value,
	hasOldModel bool,
	table any,
	tableValue reflect.Value,
) error {
	reqType := reqValue.Type()
	for i := 0; i < reqValue.NumField(); i++ {
		structField := reqType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		fieldValue := reqValue.Field(i)
		if marker, ok := patchFieldValidationFromValue(fieldValue); ok {
			if err := appendBuildGenUpdate(updates, marker, structField, oldValue, hasOldModel, table, tableValue); err != nil {
				return err
			}
			continue
		}

		if !structField.Anonymous {
			continue
		}

		nestedValue, ok := patchBuildNestedStructField(fieldValue)
		if !ok {
			continue
		}
		if err := buildGenUpdatesFromStruct(updates, nestedValue, oldValue, hasOldModel, table, tableValue); err != nil {
			return err
		}
	}
	return nil
}

func appendBuildGenUpdate(
	updates *[]field.AssignExpr,
	marker patchFieldValidationMarker,
	structField reflect.StructField,
	oldValue reflect.Value,
	hasOldModel bool,
	table any,
	tableValue reflect.Value,
) error {
	if !marker.patchFieldValidationSet() {
		return nil
	}

	candidates := patchFieldCandidates(structField)
	if len(candidates) == 0 {
		return nil
	}

	target, err := patchFindTargetValue(table, tableValue, candidates)
	if err != nil {
		return fmt.Errorf("字段 %s 构建更新表达式失败: %w", structField.Name, err)
	}

	fieldOldValue, err := patchFindOldValue(oldValue, hasOldModel, candidates)
	if err != nil {
		return fmt.Errorf("字段 %s 构建更新表达式失败: %w", structField.Name, err)
	}

	assignExpr, changed, err := patchBuildAssignExpr(marker, fieldOldValue, target)
	if err != nil {
		return fmt.Errorf("字段 %s 构建更新表达式失败: %w", structField.Name, err)
	}
	if changed {
		*updates = append(*updates, assignExpr)
	}
	return nil
}

func patchBuildAssignExpr(marker patchFieldValidationMarker, oldValue any, target any) (field.AssignExpr, bool, error) {
	if !marker.patchFieldValidationSet() {
		return nil, false, nil
	}

	oldInfo := parsePatchDynamicOldValue(oldValue)
	resolvedTarget, err := resolvePatchDynamicTarget(target)
	if err != nil {
		return nil, false, err
	}

	if !oldInfo.known {
		if marker.patchFieldValidationNull() {
			if resolvedTarget.setNull == nil {
				return nil, false, errors.New("目标字段不支持 Null()")
			}
			assignExpr, err := resolvedTarget.setNull()
			if err != nil {
				return nil, false, err
			}
			return assignExpr, true, nil
		}
		assignExpr, err := resolvedTarget.setValue(marker.patchFieldValidationValue())
		if err != nil {
			return nil, false, err
		}
		return assignExpr, true, nil
	}

	if oldInfo.nullable {
		if marker.patchFieldValidationNull() {
			if oldInfo.isNull {
				return nil, false, nil
			}
			if resolvedTarget.setNull == nil {
				return nil, false, errors.New("目标字段不支持 Null()")
			}
			assignExpr, err := resolvedTarget.setNull()
			if err != nil {
				return nil, false, err
			}
			return assignExpr, true, nil
		}

		newValue := marker.patchFieldValidationValue()
		if !oldInfo.isNull && reflect.DeepEqual(oldInfo.value, newValue) {
			return nil, false, nil
		}
		assignExpr, err := resolvedTarget.setValue(newValue)
		if err != nil {
			return nil, false, err
		}
		return assignExpr, true, nil
	}

	if marker.patchFieldValidationNull() {
		return nil, false, nil
	}

	newValue := marker.patchFieldValidationValue()
	if reflect.DeepEqual(oldInfo.value, newValue) {
		return nil, false, nil
	}

	assignExpr, err := resolvedTarget.setValue(newValue)
	if err != nil {
		return nil, false, err
	}
	return assignExpr, true, nil
}

func resolvePatchDynamicTarget(target any) (patchResolvedDynamicTarget, error) {
	value := reflect.ValueOf(target)
	if !value.IsValid() {
		return patchResolvedDynamicTarget{}, errors.New("target 不能为空")
	}

	valueMethod := patchLookupMethod(value, "Value")
	if !valueMethod.IsValid() {
		return patchResolvedDynamicTarget{}, fmt.Errorf("target 类型不支持 Value: %T", target)
	}

	resolved := patchResolvedDynamicTarget{
		setValue: func(arg any) (field.AssignExpr, error) {
			return callPatchDynamicMethod(valueMethod, arg)
		},
	}

	nullMethod := patchLookupMethod(value, "Null")
	if nullMethod.IsValid() {
		resolved.setNull = func() (field.AssignExpr, error) {
			return callPatchDynamicMethod(nullMethod)
		}
	}

	return resolved, nil
}

func callPatchDynamicMethod(method reflect.Value, args ...any) (field.AssignExpr, error) {
	if !method.IsValid() {
		return nil, errors.New("method 不能为空")
	}

	methodType := method.Type()
	if methodType.NumIn() != len(args) || methodType.NumOut() != 1 {
		return nil, errors.New("方法签名不合法")
	}

	callArgs := make([]reflect.Value, 0, len(args))
	for i, arg := range args {
		converted, err := patchConvertDynamicArgument(arg, methodType.In(i))
		if err != nil {
			return nil, err
		}
		callArgs = append(callArgs, converted)
	}

	result := method.Call(callArgs)
	if len(result) != 1 {
		return nil, errors.New("方法返回值不合法")
	}
	return patchConvertAssignExpr(result[0])
}

func patchConvertDynamicArgument(arg any, targetType reflect.Type) (reflect.Value, error) {
	if arg == nil {
		if patchCanNilType(targetType) {
			return reflect.Zero(targetType), nil
		}
		return reflect.Value{}, fmt.Errorf("参数类型不匹配: need=%s got=nil", targetType)
	}

	value := reflect.ValueOf(arg)
	if value.Type().AssignableTo(targetType) {
		return value, nil
	}
	if value.Type().ConvertibleTo(targetType) {
		return value.Convert(targetType), nil
	}
	if value.CanAddr() && value.Addr().Type().AssignableTo(targetType) {
		return value.Addr(), nil
	}
	if targetType.Kind() == reflect.Interface {
		if value.Type().Implements(targetType) {
			return value, nil
		}
		if reflect.PointerTo(value.Type()).Implements(targetType) {
			ptr := reflect.New(value.Type())
			ptr.Elem().Set(value)
			return ptr, nil
		}
	}
	return reflect.Value{}, fmt.Errorf("参数类型不匹配: need=%s got=%s", targetType, value.Type())
}

func patchConvertAssignExpr(value reflect.Value) (field.AssignExpr, error) {
	if !value.IsValid() {
		return nil, errors.New("返回值不能为空")
	}
	if !value.Type().Implements(patchAssignExprType) {
		return nil, fmt.Errorf("返回值必须实现 field.AssignExpr，当前类型=%s", value.Type())
	}
	assignExpr, ok := value.Interface().(field.AssignExpr)
	if !ok {
		return nil, errors.New("返回值转换 field.AssignExpr 失败")
	}
	return assignExpr, nil
}

func patchLookupMethod(value reflect.Value, methodName string) reflect.Value {
	if !value.IsValid() {
		return reflect.Value{}
	}
	if method := value.MethodByName(methodName); method.IsValid() {
		return method
	}
	if value.CanAddr() {
		if method := value.Addr().MethodByName(methodName); method.IsValid() {
			return method
		}
	}
	if value.Kind() != reflect.Ptr {
		ptr := reflect.New(value.Type())
		ptr.Elem().Set(value)
		if method := ptr.MethodByName(methodName); method.IsValid() {
			return method
		}
	}
	return reflect.Value{}
}

func parsePatchDynamicOldValue(oldValue any) patchDynamicOldValueInfo {
	if _, ok := oldValue.(patchUnknownOldValue); ok {
		return patchDynamicOldValueInfo{}
	}
	if oldValue == nil {
		return patchDynamicOldValueInfo{
			known:    true,
			nullable: true,
			isNull:   true,
		}
	}

	value := reflect.ValueOf(oldValue)
	for value.IsValid() && value.Kind() == reflect.Interface {
		if value.IsNil() {
			return patchDynamicOldValueInfo{
				known:    true,
				nullable: true,
				isNull:   true,
			}
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return patchDynamicOldValueInfo{
			known:    true,
			nullable: true,
			isNull:   true,
		}
	}

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return patchDynamicOldValueInfo{
				known:    true,
				nullable: true,
				isNull:   true,
			}
		}
		return patchDynamicOldValueInfo{
			known:    true,
			nullable: true,
			value:    value.Elem().Interface(),
		}
	}

	return patchDynamicOldValueInfo{
		known: true,
		value: value.Interface(),
	}
}

func patchBuildStructValue(value any, name string) (reflect.Value, error) {
	if value == nil {
		return reflect.Value{}, fmt.Errorf("%s 不能为空", name)
	}

	result := reflect.ValueOf(value)
	for result.IsValid() && result.Kind() == reflect.Ptr {
		if result.IsNil() {
			return reflect.Value{}, fmt.Errorf("%s 不能为空", name)
		}
		result = result.Elem()
	}
	if !result.IsValid() || result.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("%s 必须是结构体或结构体指针", name)
	}
	return result, nil
}

func patchBuildOptionalStructValue(value any) (reflect.Value, bool, error) {
	if value == nil {
		return reflect.Value{}, false, nil
	}

	result := reflect.ValueOf(value)
	for result.IsValid() && result.Kind() == reflect.Interface {
		if result.IsNil() {
			return reflect.Value{}, false, nil
		}
		result = result.Elem()
	}
	for result.IsValid() && result.Kind() == reflect.Ptr {
		if result.IsNil() {
			return reflect.Value{}, false, nil
		}
		result = result.Elem()
	}
	if !result.IsValid() {
		return reflect.Value{}, false, nil
	}
	if result.Kind() != reflect.Struct {
		return reflect.Value{}, false, errors.New("oldModel 必须是结构体、结构体指针或 nil")
	}
	return result, true, nil
}

func patchBuildNestedStructField(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return value, true
}

func patchFieldCandidates(structField reflect.StructField) []string {
	names := make([]string, 0, 3)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || name == "-" {
			return
		}
		for _, existing := range names {
			if existing == name {
				return
			}
		}
		names = append(names, name)
	}

	patchName := strings.TrimSpace(strings.SplitN(structField.Tag.Get(patchTag), ",", 2)[0])
	if patchName == "-" {
		return nil
	}
	add(patchName)
	add(structField.Name)
	add(strings.SplitN(structField.Tag.Get(TagJSON), ",", 2)[0])
	return names
}

func patchFindTargetValue(table any, tableValue reflect.Value, candidates []string) (any, error) {
	if fieldValue, ok := patchFindStructFieldValue(tableValue, candidates); ok {
		return fieldValue.Interface(), nil
	}

	getter, ok := table.(patchGetFieldByName)
	if ok {
		for _, candidate := range candidates {
			if target, exists := getter.GetFieldByName(candidate); exists {
				return target, nil
			}
		}
	}

	return nil, fmt.Errorf("在 table 中找不到字段: %s", strings.Join(candidates, "/"))
}

func patchFindOldValue(oldValue reflect.Value, hasOldModel bool, candidates []string) (any, error) {
	if !hasOldModel {
		return patchUnknownOldValue{}, nil
	}

	fieldValue, ok := patchFindStructFieldValue(oldValue, candidates)
	if !ok {
		return nil, fmt.Errorf("在 oldModel 中找不到字段: %s", strings.Join(candidates, "/"))
	}
	return fieldValue.Interface(), nil
}

func patchFindStructFieldValue(value reflect.Value, candidates []string) (reflect.Value, bool) {
	for _, candidate := range candidates {
		if fieldValue, ok := patchFindStructFieldByName(value, candidate); ok {
			return fieldValue, true
		}
	}
	for _, candidate := range candidates {
		if fieldValue, ok := patchFindStructFieldByJSONTag(value, candidate); ok {
			return fieldValue, true
		}
	}
	return reflect.Value{}, false
}

func patchFindStructFieldByName(value reflect.Value, target string) (reflect.Value, bool) {
	value = patchDereferenceStructValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		fieldValue := value.Field(i)
		if structField.Name == target {
			return fieldValue, true
		}
		if structField.Anonymous {
			if nested, ok := patchFindStructFieldByName(fieldValue, target); ok {
				return nested, true
			}
		}
	}
	return reflect.Value{}, false
}

func patchFindStructFieldByJSONTag(value reflect.Value, target string) (reflect.Value, bool) {
	value = patchDereferenceStructValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		structField := valueType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		fieldValue := value.Field(i)
		if strings.SplitN(structField.Tag.Get(TagJSON), ",", 2)[0] == target {
			return fieldValue, true
		}
		if structField.Anonymous {
			if nested, ok := patchFindStructFieldByJSONTag(fieldValue, target); ok {
				return nested, true
			}
		}
	}
	return reflect.Value{}, false
}

func patchDereferenceStructValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func patchCanNilType(targetType reflect.Type) bool {
	switch targetType.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return true
	default:
		return false
	}
}
