package wd

import "reflect"

type excelStructFieldMeta struct {
	index       int
	name        string
	tag         string
	columnTitle string
	fieldType   reflect.Type
	isPointer   bool
	ok          bool
}

func parseExcelStructField(field reflect.StructField, index int) excelStructFieldMeta {
	tagOptions := parseExcelTag(field.Tag.Get(TagExcel))
	if !tagOptions.ok {
		return excelStructFieldMeta{}
	}

	fieldType := field.Type
	isPointer := fieldType.Kind() == reflect.Ptr
	if isPointer {
		fieldType = fieldType.Elem()
	}

	return excelStructFieldMeta{
		index:       index,
		name:        field.Name,
		tag:         tagOptions.columnName,
		columnTitle: tagOptions.columnTitle,
		fieldType:   fieldType,
		isPointer:   isPointer,
		ok:          true,
	}
}
