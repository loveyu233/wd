package wd

import (
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

// GenNewTimeIsDateOnly 判断column这个列的值是否是dateTime[0]这个日期
func GenNewTimeIsDateOnly(table schema.Tabler, column field.IColumnName, dateTime ...DateOnly) field.Expr {
	if len(dateTime) == 0 {
		dateTime = append(dateTime, NowAsDateOnly())
	}
	return field.NewTime(table.TableName(), column.ColumnName().String()).Date().Eq(dateTime[0].Time())
}

// GenNewUnsafeFieldRaw 用来创建原始 SQL 字段引用。
func GenNewUnsafeFieldRaw(rawSQL string, vars ...interface{}) field.Field {
	return field.NewUnsafeFieldRaw(rawSQL, vars...)
}
