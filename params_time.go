package wd

import (
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type ReqDateTimeStartEnd struct {
	StartDateTime string `json:"start_date_time" form:"start_date_time"`
	EndDateTime   string `json:"end_date_time" form:"end_date_time"`

	startDateTime DateTime
	endDateTime   DateTime
}

func (req *ReqDateTimeStartEnd) GetStartAndEndDateTime() (start DateTime, end DateTime, err error) {
	if err := req.Parse(); err != nil {
		return DateTime{}, DateTime{}, err
	}
	return req.startDateTime, req.endDateTime, nil
}

// Parse 用来解析开始与结束的日期时间并设置过滤标记。
func (req *ReqDateTimeStartEnd) Parse() error {
	var err error
	req.startDateTime, err = parseOptional(req.StartDateTime, ParseDateTimeValue)
	if err != nil {
		return err
	}
	req.endDateTime, err = parseOptional(req.EndDateTime, ParseDateTimeValue)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断是否具备完整的时间范围过滤条件。
func (req *ReqDateTimeStartEnd) Enabled() bool {
	return req.StartDateTime != "" && req.EndDateTime != ""
}

func (req *ReqDateTimeStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.startDateTime),
		filed.Lte(req.endDateTime),
	}, nil
}

type ReqDateTime struct {
	DateTime string `json:"date_time" form:"date_time"`
	dateTime DateTime
}

func (req *ReqDateTime) GetDateTime() (DateTime, error) {
	if err := req.Parse(); err != nil {
		return DateTime{}, err
	}
	return req.dateTime, nil
}

// Parse 用来解析单个日期时间字符串。
func (req *ReqDateTime) Parse() error {
	if req.DateTime != "" {
		s, err := ParseDateTimeValue(req.DateTime)
		if err != nil {
			return err
		}
		req.dateTime = s
	}
	return nil
}
func (req *ReqDateTime) Enabled() bool {
	return req.DateTime != ""
}

func (req *ReqDateTime) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Eq(req.dateTime),
	}, nil
}

type ReqDateStartEnd struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`

	startDate DateOnly
	endDate   DateOnly
}

func (req *ReqDateStartEnd) GetStartAndEndDate() (DateOnly, DateOnly, bool) {
	if err := req.Parse(); err != nil {
		return DateOnly{}, DateOnly{}, false
	}
	return req.startDate, req.endDate, req.Enabled()
}

// Parse 用来解析日期范围参数。
func (req *ReqDateStartEnd) Parse() error {
	var err error
	req.startDate, err = parseOptional(req.StartDate, ParseDateOnly)
	if err != nil {
		return err
	}
	req.endDate, err = parseOptional(req.EndDate, ParseDateOnly)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断是否启用日期范围过滤。
func (req *ReqDateStartEnd) Enabled() bool {
	return req.StartDate != "" && req.EndDate != ""
}

func (req *ReqDateStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.startDate),
		filed.Lte(req.endDate),
	}, nil
}

type ReqDate struct {
	Date string `json:"date" form:"date"`
	date DateOnly
}

func (req *ReqDate) GetDate() (DateOnly, error) {
	if err := req.Parse(); err != nil {
		return DateOnly{}, err
	}
	return req.date, nil
}

// Parse 用来解析单个日期字符串。
func (req *ReqDate) Parse() error {
	var err error
	req.date, err = parseOptional(req.Date, ParseDateOnly)
	if err != nil {
		return err
	}
	return nil
}
func (req *ReqDate) Enabled() bool {
	return req.Date != ""
}

func (req *ReqDate) GenWhereFilters(table schema.Tabler, filed field.Field) (gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeIsDateOnly(table, filed, req.date), nil
}

type ReqTimeStartEnd struct {
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`

	startTime TimeOnly
	endTime   TimeOnly
}

func (req *ReqTimeStartEnd) GetStartAndEndTime() (TimeOnly, TimeOnly, bool) {
	if err := req.Parse(); err != nil {
		return TimeOnly{}, TimeOnly{}, false
	}
	return req.startTime, req.endTime, req.Enabled()
}

// Parse 用来解析起止时间并设置 TimeFilter。
func (req *ReqTimeStartEnd) Parse() error {
	var err error
	req.startTime, err = parseOptional(req.StartTime, ParseTimeOnly)
	if err != nil {
		return err
	}
	req.endTime, err = parseOptional(req.EndTime, ParseTimeOnly)

	return nil
}

// Enabled 判断是否启用具体时间的范围过滤。
func (req *ReqTimeStartEnd) Enabled() bool {
	return req.StartTime != "" && req.EndTime != ""
}

func (req *ReqTimeStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.startTime),
		filed.Lte(req.endTime),
	}, nil
}

type ReqTime struct {
	Time string `json:"time" form:"time"`
	time TimeOnly
}

func (req *ReqTime) GetTime() (TimeOnly, error) {
	if err := req.Parse(); err != nil {
		return TimeOnly{}, err
	}
	return req.time, nil
}

// Parse 用来解析单个时间值。
func (req *ReqTime) Parse() error {
	var err error
	req.time, err = parseOptional(req.Time, ParseTimeOnly)
	if err != nil {
		return err
	}
	return nil
}
func (req *ReqTime) Enabled() bool {
	return req.Time != ""
}

func (req *ReqTime) GenWhereFilters(filed field.Field) (gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return filed.Eq(req.time), nil
}

type ReqTimeHourMinuteStartEnd struct {
	StartTimeHourMinute string `json:"start_time_hour_minute" form:"start_time_hour_minute"`
	EndTimeHourMinute   string `json:"end_time_hour_minute" form:"end_time_hour_minute"`

	startTimeHourMinute TimeHourMinute
	endTimeHourMinute   TimeHourMinute
}

func (req *ReqTimeHourMinuteStartEnd) GetStartAndEndTimeHourMinute() (TimeHourMinute, TimeHourMinute, bool) {
	if err := req.Parse(); err != nil {
		return TimeHourMinute{}, TimeHourMinute{}, false
	}
	return req.startTimeHourMinute, req.endTimeHourMinute, req.Enabled()
}

// Parse 用来解析起止的时分参数。
func (req *ReqTimeHourMinuteStartEnd) Parse() error {
	var err error
	req.startTimeHourMinute, err = parseOptional(req.StartTimeHourMinute, ParseHourMinute)
	if err != nil {
		return err
	}
	req.endTimeHourMinute, err = parseOptional(req.EndTimeHourMinute, ParseHourMinute)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinuteStartEnd) Enabled() bool {
	return req.StartTimeHourMinute != "" && req.EndTimeHourMinute != ""
}

func (req *ReqTimeHourMinuteStartEnd) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Gte(req.startTimeHourMinute),
		filed.Lte(req.endTimeHourMinute),
	}, nil
}

type ReqTimeHourMinute struct {
	TimeHourMinute string `json:"time_hour_minute" form:"time_hour_minute"`
	timeHourMinute TimeHourMinute
}

func (req *ReqTimeHourMinute) GetTimeHourMinute() (TimeHourMinute, error) {
	if err := req.Parse(); err != nil {
		return TimeHourMinute{}, err
	}
	return req.timeHourMinute, nil
}

// Parse 用来解析单个时分字符串。
func (req *ReqTimeHourMinute) Parse() error {
	var err error
	req.timeHourMinute, err = parseOptional(req.TimeHourMinute, ParseHourMinute)
	if err != nil {
		return err
	}

	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinute) Enabled() bool {
	return req.TimeHourMinute != ""
}

func (req *ReqTimeHourMinute) GenWhereFilters(filed field.Field) ([]gen.Condition, error) {
	if err := req.Parse(); err != nil {
		return nil, err
	}
	if !req.Enabled() {
		return nil, nil
	}
	return []gen.Condition{
		filed.Eq(req.timeHourMinute),
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
