package wd

import (
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type ReqDateTimeStartEnd struct {
	StartDateTimeStr string `json:"start_date_time" form:"start_date_time"`
	EndDateTimeStr   string `json:"end_date_time" form:"end_date_time"`

	StartDateTime DateTime `json:"-" form:"-"`
	EndDateTime   DateTime `json:"-" form:"-"`
}

// Parse 用来解析开始与结束的日期时间并设置过滤标记。
func (req *ReqDateTimeStartEnd) Parse() error {
	var err error
	req.StartDateTime, err = parseOptional(req.StartDateTimeStr, ParseDateTimeValue)
	if err != nil {
		return err
	}
	req.EndDateTime, err = parseOptional(req.EndDateTimeStr, ParseDateTimeValue)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断是否具备完整的时间范围过滤条件。
func (req *ReqDateTimeStartEnd) Enabled() bool {
	return req.StartDateTimeStr != "" && req.EndDateTimeStr != ""
}

func (req *ReqDateTimeStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.StartDateTime),
		filed.Lte(req.EndDateTime),
	}, nil
}

type ReqDateTime struct {
	DateTimeStr string   `json:"date_time" form:"date_time"`
	DateTime    DateTime `json:"-" form:"-"`
}

// Parse 用来解析单个日期时间字符串。
func (req *ReqDateTime) Parse() error {
	if req.DateTimeStr != "" {
		s, err := ParseDateTimeValue(req.DateTimeStr)
		if err != nil {
			return err
		}
		req.DateTime = s
	}
	return nil
}
func (req *ReqDateTime) Enabled() bool {
	return req.DateTimeStr != ""
}

func (req *ReqDateTime) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Eq(req.DateTime),
	}, nil
}

type ReqDateStartEnd struct {
	StartDateStr string `json:"start_date" form:"start_date"`
	EndDateStr   string `json:"end_date" form:"end_date"`

	StartDate DateOnly `json:"-" form:"-"`
	EndDate   DateOnly `json:"-" form:"-"`
}

// Parse 用来解析日期范围参数。
func (req *ReqDateStartEnd) Parse() error {
	var err error
	req.StartDate, err = parseOptional(req.StartDateStr, ParseDateOnly)
	if err != nil {
		return err
	}
	req.EndDate, err = parseOptional(req.EndDateStr, ParseDateOnly)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断是否启用日期范围过滤。
func (req *ReqDateStartEnd) Enabled() bool {
	return req.StartDateStr != "" && req.EndDateStr != ""
}

func (req *ReqDateStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.StartDate),
		filed.Lte(req.EndDate),
	}, nil
}

type ReqDate struct {
	DateStr string   `json:"date" form:"date"`
	Date    DateOnly `json:"-" form:"-"`
}

// Parse 用来解析单个日期字符串。
func (req *ReqDate) Parse() error {
	var err error
	req.Date, err = parseOptional(req.DateStr, ParseDateOnly)
	if err != nil {
		return err
	}
	return nil
}
func (req *ReqDate) Enabled() bool {
	return req.DateStr != ""
}

func (req *ReqDate) GenWhereFilters(table schema.Tabler, filed field.Field) (gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeIsDateOnly(table, filed, req.Date), nil
}

type ReqTimeStartEnd struct {
	StartTimeStr string `json:"start_time" form:"start_time"`
	EndTimeStr   string `json:"end_time" form:"end_time"`

	StartTime TimeOnly `json:"-" form:"-"`
	EndTime   TimeOnly `json:"-" form:"-"`
}

// Parse 用来解析起止时间并设置 TimeFilter。
func (req *ReqTimeStartEnd) Parse() error {
	var err error
	req.StartTime, err = parseOptional(req.StartTimeStr, ParseTimeOnly)
	if err != nil {
		return err
	}
	req.EndTime, err = parseOptional(req.EndTimeStr, ParseTimeOnly)

	return nil
}

// Enabled 判断是否启用具体时间的范围过滤。
func (req *ReqTimeStartEnd) Enabled() bool {
	return req.StartTimeStr != "" && req.EndTimeStr != ""
}

func (req *ReqTimeStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.StartTime),
		filed.Lte(req.EndTime),
	}, nil
}

type ReqTime struct {
	TimeStr string   `json:"time" form:"time"`
	Time    TimeOnly `json:"-" form:"-"`
}

// Parse 用来解析单个时间值。
func (req *ReqTime) Parse() error {
	var err error
	req.Time, err = parseOptional(req.TimeStr, ParseTimeOnly)
	if err != nil {
		return err
	}
	return nil
}
func (req *ReqTime) Enabled() bool {
	return req.TimeStr != ""
}

func (req *ReqTime) GenWhereFilters(filed field.Field) (gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return filed.Eq(req.Time), nil
}

type ReqTimeHourMinuteStartEnd struct {
	StartTimeHourMinuteStr string `json:"start_time_hour_minute" form:"start_time_hour_minute"`
	EndTimeHourMinuteStr   string `json:"end_time_hour_minute" form:"end_time_hour_minute"`

	StartTimeHourMinute TimeHourMinute `json:"-" form:"-"`
	EndTimeHourMinute   TimeHourMinute `json:"-" form:"-"`
}

// Parse 用来解析起止的时分参数。
func (req *ReqTimeHourMinuteStartEnd) Parse() error {
	var err error
	req.StartTimeHourMinute, err = parseOptional(req.StartTimeHourMinuteStr, ParseHourMinute)
	if err != nil {
		return err
	}
	req.EndTimeHourMinute, err = parseOptional(req.EndTimeHourMinuteStr, ParseHourMinute)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinuteStartEnd) Enabled() bool {
	return req.StartTimeHourMinuteStr != "" && req.EndTimeHourMinuteStr != ""
}

func (req *ReqTimeHourMinuteStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.StartTimeHourMinute),
		filed.Lte(req.EndTimeHourMinute),
	}, nil
}

type ReqTimeHourMinute struct {
	TimeHourMinuteStr string         `json:"time_hour_minute" form:"time_hour_minute"`
	TimeHourMinute    TimeHourMinute `json:"-" form:"-"`
}

// Parse 用来解析单个时分字符串。
func (req *ReqTimeHourMinute) Parse() error {
	var err error
	req.TimeHourMinute, err = parseOptional(req.TimeHourMinuteStr, ParseHourMinute)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinute) Enabled() bool {
	return req.TimeHourMinuteStr != ""
}

func (req *ReqTimeHourMinute) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Eq(req.TimeHourMinute),
	}, nil
}

func parseOptional[T any](raw string, parse func(string) (T, error)) (value T, err error) {
	if raw == "" {
		return value, nil
	}
	parsed, err := parse(raw)
	if err != nil {
		var zero T
		return zero, err
	}
	return parsed, nil
}
