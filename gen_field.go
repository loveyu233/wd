package wd

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

// GenJSONArrayQuery 用来针对 JSON 数组列构建查询表达式。
func GenJSONArrayQuery(column field.IColumnName) *datatypes.JSONArrayExpression {
	return datatypes.JSONArrayQuery(column.ColumnName().String())
}

// GenJSONArrayQueryContainsValue 判断column这个列是否包含value这个值
func GenJSONArrayQueryContainsValue(column field.IColumnName, value string) []gen.Condition {
	return gen.Cond(datatypes.JSONArrayQuery(column.ColumnName().String()).Contains(value))
}

// GenNewTime 用来为指定表字段创建时间字段对象。
func GenNewTime(table schema.Tabler, column field.IColumnName) field.Time {
	return field.NewTime(table.TableName(), column.ColumnName().String())
}

// GenCustomTimeBetween 对column这个时间字段进行快捷的Between查询
func GenCustomTimeBetween(table schema.Tabler, column field.IColumnName, left, right CustomTime) field.Expr {
	leftCustomTimeType := left.Type()
	rightCustomTimeType := right.Type()
	if leftCustomTimeType != rightCustomTimeType {
		panic("leftCustomTimeType != rightCustomTimeType")
	}

	switch leftCustomTimeType {
	case "date_time":
		return GenNewTime(table, column).Between(left.Time(), right.Time())
	case "date_only":
		return GenNewTime(table, column).Date().Between(left.Time(), right.Time())
	case "time_only", "time_hour_minute":
		return GenNewUnsafeFieldRaw(fmt.Sprintf("TIME(%s.%s) between ? and ?", table.TableName(), column.ColumnName().String()), left, right)
	default:
		return nil
	}
}

// GenCustomTimeEq 判断column这个列的值是否是dateTime这个日期
func GenCustomTimeEq(table schema.Tabler, column field.IColumnName, dateTime CustomTime) field.Expr {
	customTimeType := dateTime.Type()

	switch customTimeType {
	case "date_time":
		return GenNewTime(table, column).Eq(dateTime.Time())
	case "date_only":
		return GenNewTime(table, column).Date().Eq(dateTime.Time())
	case "time_only":
		return GenNewUnsafeFieldRaw(fmt.Sprintf("TIME(%s.%s) = ?", table.TableName(), column.ColumnName().String()), dateTime)
	case "time_hour_minute":
		return GenNewUnsafeFieldRaw(fmt.Sprintf("TIME(%s.%s) = ?", table.TableName(), column.ColumnName().String()), fmt.Sprintf("%s:00", dateTime))
	default:
		return nil
	}
}

// GenNewUnsafeFieldRaw 用来创建原始 SQL 字段引用。
func GenNewUnsafeFieldRaw(rawSQL string, vars ...interface{}) field.Field {
	return field.NewUnsafeFieldRaw(rawSQL, vars...)
}

// GenNewBetween 支持的类型有：uint | uint8 | uint16 | uint32 | uint64 | int | int8 | int32 | int64 | float32 | float64 | decimal
func GenNewBetween[T uint | uint8 | uint16 | uint32 | uint64 | int | int8 | int32 | int64 | float32 | float64 | decimal.Decimal](table schema.Tabler, column field.IColumnName, left, right T) field.Expr {
	tableName := table.TableName()
	columnName := column.ColumnName().String()
	switch leftVal := any(left).(type) {
	case uint:
		rightVal := any(right).(uint)
		return field.NewUint(tableName, columnName).Between(leftVal, rightVal)
	case uint8:
		rightVal := any(right).(uint8)
		return field.NewUint8(tableName, columnName).Between(leftVal, rightVal)
	case uint16:
		rightVal := any(right).(uint16)
		return field.NewUint16(tableName, columnName).Between(leftVal, rightVal)
	case uint32:
		rightVal := any(right).(uint32)
		return field.NewUint32(tableName, columnName).Between(leftVal, rightVal)
	case uint64:
		rightVal := any(right).(uint64)
		return field.NewUint64(tableName, columnName).Between(leftVal, rightVal)
	case int:
		rightVal := any(right).(int)
		return field.NewInt(tableName, columnName).Between(leftVal, rightVal)
	case int8:
		rightVal := any(right).(int8)
		return field.NewInt8(tableName, columnName).Between(leftVal, rightVal)
	case int32:
		rightVal := any(right).(int32)
		return field.NewInt32(tableName, columnName).Between(leftVal, rightVal)
	case int64:
		rightVal := any(right).(int64)
		return field.NewInt64(tableName, columnName).Between(leftVal, rightVal)
	case float32:
		rightVal := any(right).(float32)
		return field.NewFloat32(tableName, columnName).Between(leftVal, rightVal)
	case float64:
		rightVal := any(right).(float64)
		return field.NewFloat64(tableName, columnName).Between(leftVal, rightVal)
	case decimal.Decimal:
		rightVal := any(right).(decimal.Decimal)
		return GenNewUnsafeFieldRaw(fmt.Sprintf("%s.%s >= ? and %s.%s <= ?", tableName, columnName, tableName, columnName), leftVal, rightVal)
	}
	return nil
}
