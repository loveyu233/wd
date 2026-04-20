package wd

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Cast 用来把任意值转换为目标类型。
// 当前明确支持以下目标类型：
// - 基础标量：string、bool、各类 int/uint、float32/float64
// - 业务类型：decimal.Decimal、DateTime、DateOnly、MonthDay、TimeOnly、TimeHM
// - 反射兜底：对可赋值或可转换的命名类型/别名类型生效
//
// 这不是一个“万能 cast”工具，当前有意保持以下边界：
// - 不自动解引用指针
// - 负数不会转换为无符号整数
// - 超范围数值窄化会返回错误，而不是静默截断
// - 空白字符串对数值/金额/时间类型会返回错误
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
	case decimal.Decimal:
		return castTarget[T](castDecimal(value))
	case DateTime:
		return castTarget[T](castDateTime(value))
	case DateOnly:
		return castTarget[T](castDateOnly(value))
	case MonthDay:
		return castTarget[T](castMonthDay(value))
	case TimeOnly:
		return castTarget[T](castTimeOnly(value))
	case TimeHM:
		return castTarget[T](castTimeHM(value))
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
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && (i64 < math.MinInt32 || i64 > math.MaxInt32) {
		return 0, fmt.Errorf("值 %d 超出 int 范围", i64)
	}
	return int(i64), nil
}

func castInt8(value any) (int8, error) {
	i64, err := castInt64(value)
	if err != nil {
		return 0, err
	}
	if i64 < math.MinInt8 || i64 > math.MaxInt8 {
		return 0, fmt.Errorf("值 %d 超出 int8 范围", i64)
	}
	return int8(i64), nil
}

func castInt16(value any) (int16, error) {
	i64, err := castInt64(value)
	if err != nil {
		return 0, err
	}
	if i64 < math.MinInt16 || i64 > math.MaxInt16 {
		return 0, fmt.Errorf("值 %d 超出 int16 范围", i64)
	}
	return int16(i64), nil
}

func castInt32(value any) (int32, error) {
	i64, err := castInt64(value)
	if err != nil {
		return 0, err
	}
	if i64 < math.MinInt32 || i64 > math.MaxInt32 {
		return 0, fmt.Errorf("值 %d 超出 int32 范围", i64)
	}
	return int32(i64), nil
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
	if err != nil {
		return 0, err
	}
	if strconv.IntSize == 32 && u64 > math.MaxUint32 {
		return 0, fmt.Errorf("值 %d 超出 uint 范围", u64)
	}
	return uint(u64), nil
}

func castUint8(value any) (uint8, error) {
	u64, err := castUint64(value)
	if err != nil {
		return 0, err
	}
	if u64 > math.MaxUint8 {
		return 0, fmt.Errorf("值 %d 超出 uint8 范围", u64)
	}
	return uint8(u64), nil
}

func castUint16(value any) (uint16, error) {
	u64, err := castUint64(value)
	if err != nil {
		return 0, err
	}
	if u64 > math.MaxUint16 {
		return 0, fmt.Errorf("值 %d 超出 uint16 范围", u64)
	}
	return uint16(u64), nil
}

func castUint32(value any) (uint32, error) {
	u64, err := castUint64(value)
	if err != nil {
		return 0, err
	}
	if u64 > math.MaxUint32 {
		return 0, fmt.Errorf("值 %d 超出 uint32 范围", u64)
	}
	return uint32(u64), nil
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

func castDecimal(value any) (decimal.Decimal, error) {
	switch typed := value.(type) {
	case decimal.Decimal:
		return typed, nil
	case *decimal.Decimal:
		if typed == nil {
			return decimal.Decimal{}, fmt.Errorf("无法将 <nil> 转换为 decimal.Decimal")
		}
		return decimal.Decimal{}, fmt.Errorf("无法将 %T 转换为 decimal.Decimal", value)
	case string:
		return decimal.NewFromString(strings.TrimSpace(typed))
	case []byte:
		return decimal.NewFromString(strings.TrimSpace(string(typed)))
	case json.Number:
		return decimal.NewFromString(typed.String())
	case fmt.Stringer:
		return decimal.NewFromString(strings.TrimSpace(typed.String()))
	default:
		f64, err := numericToFloat64(value)
		if err != nil {
			return decimal.Decimal{}, fmt.Errorf("无法将 %T 转换为 decimal.Decimal", value)
		}
		return decimal.NewFromFloat(f64), nil
	}
}

func castDateTime(value any) (DateTime, error) {
	switch typed := value.(type) {
	case DateTime:
		return typed, nil
	case time.Time:
		return ToDateTime(typed), nil
	case string:
		return ParseDateTimeValue(strings.TrimSpace(typed))
	case []byte:
		return ParseDateTimeValue(strings.TrimSpace(string(typed)))
	default:
		return DateTime{}, fmt.Errorf("无法将 %T 转换为 DateTime", value)
	}
}

func castDateOnly(value any) (DateOnly, error) {
	switch typed := value.(type) {
	case DateOnly:
		return typed, nil
	case time.Time:
		return ToDateOnly(typed), nil
	case DateTime:
		return typed.ToDateOnly(), nil
	case MonthDay:
		return typed.ToDateOnly(), nil
	case string:
		return ParseDateOnly(strings.TrimSpace(typed))
	case []byte:
		return ParseDateOnly(strings.TrimSpace(string(typed)))
	default:
		return DateOnly{}, fmt.Errorf("无法将 %T 转换为 DateOnly", value)
	}
}

func castMonthDay(value any) (MonthDay, error) {
	switch typed := value.(type) {
	case MonthDay:
		return typed, nil
	case time.Time:
		return ToMonthDay(typed), nil
	case DateTime:
		return ToMonthDay(typed.Time()), nil
	case DateOnly:
		return ToMonthDay(typed.Time()), nil
	case string:
		return ParseMonthDay(strings.TrimSpace(typed))
	case []byte:
		return ParseMonthDay(strings.TrimSpace(string(typed)))
	default:
		return MonthDay{}, fmt.Errorf("无法将 %T 转换为 MonthDay", value)
	}
}

func castTimeOnly(value any) (TimeOnly, error) {
	switch typed := value.(type) {
	case TimeOnly:
		return typed, nil
	case time.Time:
		return ToTimeOnly(typed), nil
	case DateTime:
		return typed.ToTimeOnly(), nil
	case TimeHM:
		return typed.ToTimeOnly(), nil
	case string:
		return ParseTimeOnly(strings.TrimSpace(typed))
	case []byte:
		return ParseTimeOnly(strings.TrimSpace(string(typed)))
	default:
		return TimeOnly{}, fmt.Errorf("无法将 %T 转换为 TimeOnly", value)
	}
}

func castTimeHM(value any) (TimeHM, error) {
	switch typed := value.(type) {
	case TimeHM:
		return typed, nil
	case time.Time:
		return ToTimeHM(typed), nil
	case DateTime:
		return typed.ToTimeHM(), nil
	case TimeOnly:
		return typed.ToTimeHM(), nil
	case string:
		return ParseTimeHM(strings.TrimSpace(typed))
	case []byte:
		return ParseTimeHM(strings.TrimSpace(string(typed)))
	default:
		return TimeHM{}, fmt.Errorf("无法将 %T 转换为 TimeHM", value)
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
		if typed > math.MaxInt64 {
			return 0, fmt.Errorf("值 %d 超出 int64 范围", typed)
		}
		return int64(typed), nil
	case float32:
		if typed < math.MinInt64 || typed > math.MaxInt64 {
			return 0, fmt.Errorf("值 %v 超出 int64 范围", typed)
		}
		return int64(typed), nil
	case float64:
		if typed < math.MinInt64 || typed > math.MaxInt64 {
			return 0, fmt.Errorf("值 %v 超出 int64 范围", typed)
		}
		return int64(typed), nil
	default:
		return 0, fmt.Errorf("无法将 %T 转换为 int64", value)
	}
}

func numericToUint64(value any) (uint64, error) {
	switch typed := value.(type) {
	case int:
		if typed < 0 {
			return 0, fmt.Errorf("值 %d 不能转换为 uint64", typed)
		}
		return uint64(typed), nil
	case int8:
		if typed < 0 {
			return 0, fmt.Errorf("值 %d 不能转换为 uint64", typed)
		}
		return uint64(typed), nil
	case int16:
		if typed < 0 {
			return 0, fmt.Errorf("值 %d 不能转换为 uint64", typed)
		}
		return uint64(typed), nil
	case int32:
		if typed < 0 {
			return 0, fmt.Errorf("值 %d 不能转换为 uint64", typed)
		}
		return uint64(typed), nil
	case int64:
		if typed < 0 {
			return 0, fmt.Errorf("值 %d 不能转换为 uint64", typed)
		}
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
		if typed < 0 || typed > math.MaxUint64 {
			return 0, fmt.Errorf("值 %v 超出 uint64 范围", typed)
		}
		return uint64(typed), nil
	case float64:
		if typed < 0 || typed > math.MaxUint64 {
			return 0, fmt.Errorf("值 %v 超出 uint64 范围", typed)
		}
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
