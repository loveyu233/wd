package wd

import (
	"gorm.io/gen/field"
	"gorm.io/gorm/schema"
)

type ReqDateTimeStartEnd struct {
	StartDateTime string `json:"start_date_time" form:"start_date_time"`
	EndDateTime   string `json:"end_date_time" form:"end_date_time"`

	startDateTime DateTime
	endDateTime   DateTime
	isParse       bool
}

func (req *ReqDateTimeStartEnd) SetStartDateTime(startDateTime string) error {
	req.isParse = false
	req.StartDateTime = startDateTime
	return req.parse()
}

func (req *ReqDateTimeStartEnd) SetEndDateTime(endDateTime string) error {
	req.isParse = false
	req.EndDateTime = endDateTime
	return req.parse()
}

func (req *ReqDateTimeStartEnd) GetStartDateTime() (DateTime, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateTime{}, err
		}
	}
	return req.startDateTime, nil
}

func (req *ReqDateTimeStartEnd) GetEndDateTime() (DateTime, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateTime{}, err
		}
	}
	return req.endDateTime, nil
}

// Parse 用来解析开始与结束的日期时间并设置过滤标记。
func (req *ReqDateTimeStartEnd) parse() error {
	var err error
	req.startDateTime, err = parseOptional(req.StartDateTime, ParseDateTimeValue)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.endDateTime, err = parseOptional(req.EndDateTime, ParseDateTimeValue)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.isParse = true

	return nil
}

// Enabled 判断是否具备完整的时间范围过滤条件。
func (req *ReqDateTimeStartEnd) Enabled() bool {
	return req.StartDateTime != "" && req.EndDateTime != ""
}

func (req *ReqDateTimeStartEnd) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeBetween(table, column, req.startDateTime.Time(), req.endDateTime.Time()), nil
}

type ReqDateTime struct {
	DateTime string `json:"date_time" form:"date_time"`

	dateTime DateTime
	isParse  bool
}

func (req *ReqDateTime) SetDateTime(dateTime string) error {
	req.isParse = false
	req.DateTime = dateTime
	return req.parse()
}

func (req *ReqDateTime) GetDateTime() (DateTime, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateTime{}, err
		}
	}
	return req.dateTime, nil
}

// Parse 用来解析单个日期时间字符串。
func (req *ReqDateTime) parse() error {
	if req.DateTime != "" {
		s, err := ParseDateTimeValue(req.DateTime)
		if err != nil {
			return MsgErrInvalidParam(err.Error())
		}
		req.dateTime = s
	}
	req.isParse = true
	return nil
}

func (req *ReqDateTime) Enabled() bool {
	return req.DateTime != ""
}

func (req *ReqDateTime) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTime(table, column).Eq(req.dateTime.Time()), nil
}

type ReqDateStartEnd struct {
	StartDate string `json:"start_date" form:"start_date"`
	EndDate   string `json:"end_date" form:"end_date"`

	startDate DateOnly
	endDate   DateOnly
	isParse   bool
}

func (req *ReqDateStartEnd) SetStartDate(startDate string) error {
	req.isParse = false
	req.StartDate = startDate
	return req.parse()
}

func (req *ReqDateStartEnd) SetEndDate(endDate string) error {
	req.isParse = false
	req.EndDate = endDate
	return req.parse()
}

func (req *ReqDateStartEnd) GetStartDate() (DateOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateOnly{}, err
		}
	}
	return req.startDate, nil
}

func (req *ReqDateStartEnd) GetEndDate() (DateOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateOnly{}, err
		}
	}
	return req.endDate, nil
}

// Parse 用来解析日期范围参数。
func (req *ReqDateStartEnd) parse() error {
	var err error
	req.startDate, err = parseOptional(req.StartDate, ParseDateOnly)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.endDate, err = parseOptional(req.EndDate, ParseDateOnly)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.isParse = true
	return nil
}

// Enabled 判断是否启用日期范围过滤。
func (req *ReqDateStartEnd) Enabled() bool {
	return req.StartDate != "" && req.EndDate != ""
}

func (req *ReqDateStartEnd) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeBetween(table, column, req.startDate.Time(), req.endDate.Time()), nil
}

type ReqDate struct {
	Date string `json:"date" form:"date"`

	date    DateOnly
	isParse bool
}

func (req *ReqDate) SetDate(date string) error {
	req.isParse = false
	req.Date = date
	return req.parse()
}
func (req *ReqDate) GetDate() (DateOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return DateOnly{}, err
		}
	}
	return req.date, nil
}

// Parse 用来解析单个日期字符串。
func (req *ReqDate) parse() error {
	var err error
	req.date, err = parseOptional(req.Date, ParseDateOnly)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.isParse = true
	return nil
}
func (req *ReqDate) Enabled() bool {
	return req.Date != ""
}

func (req *ReqDate) GenWhereFilters(table schema.Tabler, filed field.Field) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeIsCustomDateTime(table, filed, req.date), nil
}

type ReqTimeStartEnd struct {
	StartTime string `json:"start_time" form:"start_time"`
	EndTime   string `json:"end_time" form:"end_time"`

	startTime TimeOnly
	endTime   TimeOnly
	isParse   bool
}

func (req *ReqTimeStartEnd) SetStartTime(startTime string) error {
	req.isParse = false
	req.StartTime = startTime
	return req.parse()
}

func (req *ReqTimeStartEnd) SetEndTime(endTime string) error {
	req.isParse = false
	req.EndTime = endTime
	return req.parse()
}

func (req *ReqTimeStartEnd) GetStartTime() (TimeOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeOnly{}, err
		}
	}
	return req.startTime, nil
}

func (req *ReqTimeStartEnd) GetEndTime() (TimeOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeOnly{}, err
		}
	}
	return req.endTime, nil
}

// Parse 用来解析起止时间并设置 TimeFilter。
func (req *ReqTimeStartEnd) parse() error {
	var err error
	req.startTime, err = parseOptional(req.StartTime, ParseTimeOnly)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.endTime, err = parseOptional(req.EndTime, ParseTimeOnly)
	req.isParse = true
	return nil
}

// Enabled 判断是否启用具体时间的范围过滤。
func (req *ReqTimeStartEnd) Enabled() bool {
	return req.StartTime != "" && req.EndTime != ""
}

func (req *ReqTimeStartEnd) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeBetween(table, column, req.startTime.Time(), req.endTime.Time()), nil
}

type ReqTime struct {
	Time string `json:"time" form:"time"`

	time    TimeOnly
	isParse bool
}

func (req *ReqTime) SetTime(time string) error {
	req.isParse = false
	req.Time = time
	return req.parse()
}

func (req *ReqTime) GetTime() (TimeOnly, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeOnly{}, err
		}
	}
	return req.time, nil
}

// Parse 用来解析单个时间值。
func (req *ReqTime) parse() error {
	var err error
	req.time, err = parseOptional(req.Time, ParseTimeOnly)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.isParse = true
	return nil
}
func (req *ReqTime) Enabled() bool {
	return req.Time != ""
}

func (req *ReqTime) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeIsCustomDateTime(table, column, req.time), nil
}

type ReqTimeHourMinuteStartEnd struct {
	StartTimeHourMinute string `json:"start_time_hour_minute" form:"start_time_hour_minute"`
	EndTimeHourMinute   string `json:"end_time_hour_minute" form:"end_time_hour_minute"`

	startTimeHourMinute TimeHourMinute
	endTimeHourMinute   TimeHourMinute
	isParse             bool
}

func (req *ReqTimeHourMinuteStartEnd) SetStartTimeHourMinute(startTimeHourMinute string) error {
	req.isParse = false
	req.StartTimeHourMinute = startTimeHourMinute
	return req.parse()
}

func (req *ReqTimeHourMinuteStartEnd) SetEndTimeHourMinute(endTimeHourMinute string) error {
	req.isParse = false
	req.EndTimeHourMinute = endTimeHourMinute
	return req.parse()
}

func (req *ReqTimeHourMinuteStartEnd) GetStartDateTime() (TimeHourMinute, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeHourMinute{}, err
		}
	}
	return req.startTimeHourMinute, nil
}

func (req *ReqTimeHourMinuteStartEnd) GetEndDateTime() (TimeHourMinute, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeHourMinute{}, err
		}
	}
	return req.endTimeHourMinute, nil
}

// Parse 用来解析起止的时分参数。
func (req *ReqTimeHourMinuteStartEnd) parse() error {
	var err error
	req.startTimeHourMinute, err = parseOptional(req.StartTimeHourMinute, ParseHourMinute)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.endTimeHourMinute, err = parseOptional(req.EndTimeHourMinute, ParseHourMinute)
	if err != nil {
		return err
	}
	req.isParse = true
	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinuteStartEnd) Enabled() bool {
	return req.StartTimeHourMinute != "" && req.EndTimeHourMinute != ""
}

func (req *ReqTimeHourMinuteStartEnd) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeBetween(table, column, req.startTimeHourMinute.Time(), req.endTimeHourMinute.Time()), nil
}

type ReqTimeHourMinute struct {
	TimeHourMinute string `json:"time_hour_minute" form:"time_hour_minute"`

	timeHourMinute TimeHourMinute
	isParse        bool
}

func (req *ReqTimeHourMinute) SetTimeHourMinute(timeHourMinute string) error {
	req.isParse = false
	req.TimeHourMinute = timeHourMinute
	return req.parse()
}

func (req *ReqTimeHourMinute) GetTimeHourMinute() (TimeHourMinute, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return TimeHourMinute{}, err
		}
	}
	return req.timeHourMinute, nil
}

// Parse 用来解析单个时分字符串。
func (req *ReqTimeHourMinute) parse() error {
	var err error
	req.timeHourMinute, err = parseOptional(req.TimeHourMinute, ParseHourMinute)
	if err != nil {
		return MsgErrInvalidParam(err.Error())
	}
	req.isParse = true
	return nil
}

// Enabled 判断时分范围过滤是否生效。
func (req *ReqTimeHourMinute) Enabled() bool {
	return req.TimeHourMinute != ""
}

func (req *ReqTimeHourMinute) GenWhereFilters(table schema.Tabler, column field.IColumnName) (field.Expr, error) {
	if !req.isParse {
		if err := req.parse(); err != nil {
			return nil, err
		}
	}
	if !req.Enabled() {
		return nil, nil
	}
	return GenNewTimeIsCustomDateTime(table, column, req.timeHourMinute), nil
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
