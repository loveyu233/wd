package wd

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Cast 用来把任意值转换为目标类型，当前优先覆盖工具库内部常用的基础标量类型。
func Cast[T any](value any) (T, error) {
	var zero T
	switch any(zero).(type) {
	case string:
		return castTarget[T](castString(value))
	case bool:
		return castTarget[T](castBool(value))
	case int:
		return castTarget[T](castInt(value))
	case int8:
		return castTarget[T](castInt8(value))
	case int16:
		return castTarget[T](castInt16(value))
	case int32:
		return castTarget[T](castInt32(value))
	case int64:
		return castTarget[T](castInt64(value))
	case uint:
		return castTarget[T](castUint(value))
	case uint8:
		return castTarget[T](castUint8(value))
	case uint16:
		return castTarget[T](castUint16(value))
	case uint32:
		return castTarget[T](castUint32(value))
	case uint64:
		return castTarget[T](castUint64(value))
	case float32:
		return castTarget[T](castFloat32(value))
	case float64:
		return castTarget[T](castFloat64(value))
	default:
		return castReflectValue[T](value)
	}
}

func castTarget[T any](value any, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("类型断言失败: %T -> %T", value, zero)
	}
	return typed, nil
}

func castReflectValue[T any](value any) (T, error) {
	var zero T
	if value == nil {
		return zero, fmt.Errorf("无法将 <nil> 转换为 %T", zero)
	}

	targetType := reflect.TypeOf(zero)
	sourceValue := reflect.ValueOf(value)
	if sourceValue.Type().AssignableTo(targetType) {
		return sourceValue.Interface().(T), nil
	}
	if sourceValue.Type().ConvertibleTo(targetType) {
		return sourceValue.Convert(targetType).Interface().(T), nil
	}
	return zero, fmt.Errorf("不支持的转换: %T -> %T", value, zero)
}

func castString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case json.Number:
		return typed.String(), nil
	case nil:
		return "", fmt.Errorf("无法将 <nil> 转换为 string")
	case fmt.Stringer:
		return typed.String(), nil
	default:
		return fmt.Sprint(value), nil
	}
}

func castBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typed))
	case []byte:
		return strconv.ParseBool(strings.TrimSpace(string(typed)))
	case json.Number:
		i64, err := typed.Int64()
		if err == nil {
			return i64 != 0, nil
		}
		f64, err := typed.Float64()
		if err != nil {
			return false, err
		}
		return f64 != 0, nil
	default:
		i64, err := numericToInt64(value)
		if err == nil {
			return i64 != 0, nil
		}
		f64, ferr := numericToFloat64(value)
		if ferr == nil {
			return f64 != 0, nil
		}
		return false, fmt.Errorf("无法将 %T 转换为 bool", value)
	}
}

func castInt(value any) (int, error) {
	i64, err := castInt64(value)
	return int(i64), err
}

func castInt8(value any) (int8, error) {
	i64, err := castInt64(value)
	return int8(i64), err
}

func castInt16(value any) (int16, error) {
	i64, err := castInt64(value)
	return int16(i64), err
}

func castInt32(value any) (int32, error) {
	i64, err := castInt64(value)
	return int32(i64), err
}

func castInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
	case []byte:
		return strconv.ParseInt(strings.TrimSpace(string(typed)), 10, 64)
	case json.Number:
		return typed.Int64()
	default:
		return numericToInt64(value)
	}
}

func castUint(value any) (uint, error) {
	u64, err := castUint64(value)
	return uint(u64), err
}

func castUint8(value any) (uint8, error) {
	u64, err := castUint64(value)
	return uint8(u64), err
}

func castUint16(value any) (uint16, error) {
	u64, err := castUint64(value)
	return uint16(u64), err
}

func castUint32(value any) (uint32, error) {
	u64, err := castUint64(value)
	return uint32(u64), err
}

func castUint64(value any) (uint64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseUint(strings.TrimSpace(typed), 10, 64)
	case []byte:
		return strconv.ParseUint(strings.TrimSpace(string(typed)), 10, 64)
	case json.Number:
		i64, err := typed.Int64()
		if err != nil {
			return 0, err
		}
		return uint64(i64), nil
	default:
		return numericToUint64(value)
	}
}

func castFloat32(value any) (float32, error) {
	f64, err := castFloat64(value)
	return float32(f64), err
}

func castFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	case []byte:
		return strconv.ParseFloat(strings.TrimSpace(string(typed)), 64)
	case json.Number:
		return typed.Float64()
	default:
		return numericToFloat64(value)
	}
}

func numericToInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int:
		return int64(typed), nil
	case int8:
		return int64(typed), nil
	case int16:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case int64:
		return typed, nil
	case uint:
		return int64(typed), nil
	case uint8:
		return int64(typed), nil
	case uint16:
		return int64(typed), nil
	case uint32:
		return int64(typed), nil
	case uint64:
		return int64(typed), nil
	case float32:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("无法将 %T 转换为 int64", value)
	}
}

func numericToUint64(value any) (uint64, error) {
	switch typed := value.(type) {
	case int:
		return uint64(typed), nil
	case int8:
		return uint64(typed), nil
	case int16:
		return uint64(typed), nil
	case int32:
		return uint64(typed), nil
	case int64:
		return uint64(typed), nil
	case uint:
		return uint64(typed), nil
	case uint8:
		return uint64(typed), nil
	case uint16:
		return uint64(typed), nil
	case uint32:
		return uint64(typed), nil
	case uint64:
		return typed, nil
	case float32:
		return uint64(typed), nil
	case float64:
		return uint64(typed), nil
	default:
		return 0, fmt.Errorf("无法将 %T 转换为 uint64", value)
	}
}

func numericToFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case uint:
		return float64(typed), nil
	case uint8:
		return float64(typed), nil
	case uint16:
		return float64(typed), nil
	case uint32:
		return float64(typed), nil
	case uint64:
		return float64(typed), nil
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	default:
		return 0, fmt.Errorf("无法将 %T 转换为 float64", value)
	}
}
