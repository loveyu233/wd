package wd

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ========== 类型定义 ==========

type CustomTime interface {
	Time() time.Time
	Type() string
	IsZero() bool
}

// DateTime 完整的日期时间类型 (YYYY-MM-DD HH:MM:SS)
type DateTime time.Time

func (d DateTime) Type() string {
	return "date_time"
}

// DateOnly 只有日期的类型 (YYYY-MM-DD)
type DateOnly time.Time

func (d DateOnly) Type() string {
	return "date_only"
}

// MonthDay 只有月日的类型 (MM-DD)，数据库中按固定年份保存。
type MonthDay time.Time

func (d MonthDay) Type() string {
	return "month_day"
}

// TimeOnly 只有时间的类型，包含秒 (HH:MM:SS)
type TimeOnly time.Time

func (d TimeOnly) Type() string {
	return "time_only"
}

// TimeHM 只有小时分钟的时间类型 (HH:MM)
type TimeHM time.Time

func (d TimeHM) Type() string {
	return "time_hour_minute"
}

// ========== SQL/JSON 通用辅助 ==========

func decodeOptionalJSONString(data []byte) (string, bool, error) {
	if strings.TrimSpace(string(data)) == "null" {
		return "", true, nil
	}

	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return "", false, err
	}

	value = strings.TrimSpace(value)
	if value == "" {
		return "", true, nil
	}

	return value, false, nil
}

func resetCustomTime[T customTimeType](dst *T) {
	*dst = T(time.Time{})
}

func setCustomTime[T customTimeType](dst *T, value time.Time, normalize timeNormalizer) {
	*dst = T(normalize(value))
}

func setParsedCustomTime[T customTimeType](dst *T, value string, parse stringTimeParser) error {
	parsed, err := parse(value)
	if err != nil {
		return err
	}
	*dst = T(parsed)
	return nil
}

func scanCustomTime[T customTimeType](
	dst *T,
	raw interface{},
	parse stringTimeParser,
	normalize timeNormalizer,
	kind string,
) error {
	if raw == nil {
		resetCustomTime(dst)
		return nil
	}

	switch value := raw.(type) {
	case time.Time:
		setCustomTime(dst, value, normalize)
		return nil
	case []byte:
		return setParsedCustomTime(dst, string(value), parse)
	case string:
		return setParsedCustomTime(dst, value, parse)
	default:
		return fmt.Errorf("类型转换错误：不支持的%s格式 %T", kind, raw)
	}
}

func unmarshalCustomTime[T customTimeType](dst *T, data []byte, parse stringTimeParser) error {
	value, zero, err := decodeOptionalJSONString(data)
	if err != nil {
		return err
	}
	if zero {
		resetCustomTime(dst)
		return nil
	}
	return setParsedCustomTime(dst, value, parse)
}

func unmarshalCustomTimeParam[T customTimeType](dst *T, param string, parse stringTimeParser) error {
	param = strings.TrimSpace(param)
	if param == "" {
		resetCustomTime(dst)
		return nil
	}
	return setParsedCustomTime(dst, param, parse)
}

func normalizedCustomTime[T customTimeType](value T, normalize timeNormalizer) time.Time {
	return normalize(time.Time(value))
}

func formatSQLTime[T customTimeType](value T, layout string, normalize timeNormalizer) string {
	return formatTimeValue(normalizedCustomTime(value, normalize), layout)
}

func marshalSQLTime[T customTimeType](value T, layout string, normalize timeNormalizer) ([]byte, error) {
	return json.Marshal(formatSQLTime(value, layout, normalize))
}

func valueSQLTime[T customTimeType](value T, layout string, normalize timeNormalizer) (driver.Value, error) {
	formatted := formatSQLTime(value, layout, normalize)
	if formatted == "" {
		return nil, nil
	}
	return formatted, nil
}

// ========== DateTime 序列化 ==========

// Scan 用来将数据库字段解析为 DateTime。
func (dt *DateTime) Scan(v interface{}) error {
	return scanCustomTime(dt, v, parseDateTimeString, normalizeToShanghai, "日期时间")
}

// Value 用来把 DateTime 格式化为数据库值。
func (dt DateTime) Value() (driver.Value, error) {
	return valueSQLTime(dt, CSTLayout, normalizeToShanghai)
}

// String 用来按默认格式输出日期时间字符串。
func (dt DateTime) String() string {
	return formatSQLTime(dt, CSTLayout, normalizeToShanghai)
}

// Format 用来自定义 DateTime 的格式化输出。
func (dt DateTime) Format(layout string) string {
	return formatSQLTime(dt, layout, normalizeToShanghai)
}

// MarshalJSON 用来把 DateTime 序列化为 JSON 字符串。
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return marshalSQLTime(dt, CSTLayout, normalizeToShanghai)
}

// UnmarshalJSON 用来解析 JSON 字符串到 DateTime。
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	return unmarshalCustomTime(dt, data, parseDateTimeString)
}

// UnmarshalParam 用来解析 form/query 参数到 DateTime。
func (dt *DateTime) UnmarshalParam(param string) error {
	return unmarshalCustomTimeParam(dt, param, parseDateTimeString)
}

// ========== DateOnly 序列化 ==========

// Scan 用来从数据库的各种类型读取 DateOnly。
func (d *DateOnly) Scan(v interface{}) error {
	return scanCustomTime(d, v, parseDateOnlyString, normalizeDateOnlyValue, "日期")
}

// Value 用来把 DateOnly 格式化为数据库值。
func (d DateOnly) Value() (driver.Value, error) {
	return valueSQLTime(d, CSTLayoutDate, normalizeDateOnlyValue)
}

// String 用来输出 YYYY-MM-DD 字符串。
func (d DateOnly) String() string {
	return formatSQLTime(d, CSTLayoutDate, normalizeDateOnlyValue)
}

// Format 用来自定义 DateOnly 的输出格式。
func (d DateOnly) Format(layout string) string {
	return formatSQLTime(d, layout, normalizeDateOnlyValue)
}

// MarshalJSON 用来把 DateOnly 序列化为 JSON。
func (d DateOnly) MarshalJSON() ([]byte, error) {
	return marshalSQLTime(d, CSTLayoutDate, normalizeDateOnlyValue)
}

// UnmarshalJSON 用来从 JSON 字符串解析 DateOnly。
func (d *DateOnly) UnmarshalJSON(data []byte) error {
	return unmarshalCustomTime(d, data, parseDateOnlyString)
}

// UnmarshalParam 用来解析 form/query 参数到 DateOnly。
func (d *DateOnly) UnmarshalParam(param string) error {
	return unmarshalCustomTimeParam(d, param, parseDateOnlyString)
}

// ========== MonthDay 序列化 ==========

// Scan 用来从数据库读取 MonthDay，兼容 DATE 列返回的完整日期字符串。
func (m *MonthDay) Scan(v interface{}) error {
	return scanCustomTime(m, v, parseMonthDayStoredString, normalizeMonthDayValue, "月日")
}

// Value 用来把 MonthDay 按固定年份格式化为数据库值。
func (m MonthDay) Value() (driver.Value, error) {
	return valueSQLTime(m, CSTLayoutDate, normalizeMonthDayValue)
}

// String 用来输出 MM-DD 字符串。
func (m MonthDay) String() string {
	return formatSQLTime(m, CSTLayoutMonthDay, normalizeMonthDayValue)
}

// Format 用来自定义 MonthDay 的输出格式。
func (m MonthDay) Format(layout string) string {
	return formatSQLTime(m, layout, normalizeMonthDayValue)
}

// MarshalJSON 用来把 MonthDay 序列化为 JSON。
func (m MonthDay) MarshalJSON() ([]byte, error) {
	return marshalSQLTime(m, CSTLayoutMonthDay, normalizeMonthDayValue)
}

// UnmarshalJSON 用来从 JSON 字符串解析 MonthDay。
func (m *MonthDay) UnmarshalJSON(data []byte) error {
	return unmarshalCustomTime(m, data, parseMonthDayString)
}

// UnmarshalParam 用来解析 form/query 参数到 MonthDay。
func (m *MonthDay) UnmarshalParam(param string) error {
	return unmarshalCustomTimeParam(m, param, parseMonthDayString)
}

// ========== TimeOnly 序列化 ==========

// Scan 用来将数据库值解析为 TimeOnly。
func (t *TimeOnly) Scan(v interface{}) error {
	return scanCustomTime(t, v, parseTimeOnlyString, normalizeTimeOnlyValue, "时间")
}

// Value 用来将 TimeOnly 格式化成数据库值。
func (t TimeOnly) Value() (driver.Value, error) {
	return valueSQLTime(t, CSTLayoutTime, normalizeTimeOnlyValue)
}

// String 用来输出 HH:MM:SS 形式的字符串。
func (t TimeOnly) String() string {
	return formatSQLTime(t, CSTLayoutTime, normalizeTimeOnlyValue)
}

// Format 用来自定义 TimeOnly 的字符串表示。
func (t TimeOnly) Format(layout string) string {
	return formatSQLTime(t, layout, normalizeTimeOnlyValue)
}

// MarshalJSON 用来把 TimeOnly 序列化为 JSON 文本。
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	return marshalSQLTime(t, CSTLayoutTime, normalizeTimeOnlyValue)
}

// UnmarshalJSON 用来解析 JSON 字符串到 TimeOnly。
func (t *TimeOnly) UnmarshalJSON(data []byte) error {
	return unmarshalCustomTime(t, data, parseTimeOnlyString)
}

// UnmarshalParam 用来解析 form/query 参数到 TimeOnly。
func (t *TimeOnly) UnmarshalParam(param string) error {
	return unmarshalCustomTimeParam(t, param, parseTimeOnlyString)
}

// ========== TimeHM 序列化 ==========

// Scan 用来将数据库值解析为 TimeHM。因为数据库中没有这种类型，所以使用普通时间类型来映射
func (t *TimeHM) Scan(v interface{}) error {
	return scanCustomTime(t, v, parseTimeOnlyString, normalizeTimeHMValue, "时间")
}

// Value 用来将 TimeHM 写入数据库。
func (t TimeHM) Value() (driver.Value, error) {
	return valueSQLTime(t, CSTLayoutTimeHM, normalizeTimeHMValue)
}

// String 用来输出 HH:MM 字符串。
func (t TimeHM) String() string {
	return formatSQLTime(t, CSTLayoutTimeHM, normalizeTimeHMValue)
}

// Format 用来自定义小时分钟的输出格式。
func (t TimeHM) Format(layout string) string {
	return formatSQLTime(t, layout, normalizeTimeHMValue)
}

// MarshalJSON 用来把 TimeHM 序列化为 JSON 文本。
func (t TimeHM) MarshalJSON() ([]byte, error) {
	return marshalSQLTime(t, CSTLayoutTimeHM, normalizeTimeHMValue)
}

// UnmarshalJSON 用来解析 JSON 字符串到 TimeHM。
func (t *TimeHM) UnmarshalJSON(data []byte) error {
	return unmarshalCustomTime(t, data, parseTimeHMString)
}

// UnmarshalParam 用来解析 form/query 参数到 TimeHM。
func (t *TimeHM) UnmarshalParam(param string) error {
	return unmarshalCustomTimeParam(t, param, parseTimeHMString)
}
