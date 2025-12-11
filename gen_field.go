package wd

import (
	"time"

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

// GenNewTimeBetween 对column这个时间字段进行快捷的Between查询
func GenNewTimeBetween(table schema.Tabler, column field.IColumnName, left, right time.Time) field.Expr {
	return GenNewTime(table, column).Between(left, right)
}

// GenNewTimeIsCustomDateTime 判断column这个列的值是否是dateTime这个日期
func GenNewTimeIsCustomDateTime(table schema.Tabler, column field.IColumnName, dateTime CustomTime) field.Expr {
	return GenNewTime(table, column).Eq(dateTime.Time())
}

// GenNewUnsafeFieldRaw 用来创建原始 SQL 字段引用。
func GenNewUnsafeFieldRaw(rawSQL string, vars ...interface{}) field.Field {
	return field.NewUnsafeFieldRaw(rawSQL, vars...)
}
