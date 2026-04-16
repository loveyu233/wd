package wd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

// ========== 基础配置 ==========

var (
	ShangHaiTimeLocation *time.Location
)

type timeNormalizer func(time.Time) time.Time
type stringTimeParser func(string) (time.Time, error)
type customTimeType interface {
	DateTime | DateOnly | TimeOnly | TimeHM
}
type timeValuer interface {
	Time() time.Time
}

const (
	CSTLayout                 = "2006-01-02 15:04:05"
	CSTLayoutChinese          = "2006年01月02日 15:04:05"
	CSTLayoutPoint            = "2006.01.02 15:04:05"
	CSTLayoutDate             = "2006-01-02"
	CSTLayoutDateChinese      = "2006年01月02日"
	CSTLayoutDatePoint        = "2006.01.02"
	CSTLayoutTime             = "15:04:05"
	CSTLayoutTimeHM           = "15:04"
	CSTLayoutDateHM           = "2006-01-02 15:04"
	CSTLayoutDateHMChinese    = "2006年01月02日 15:04"
	CSTLayoutDateHMPoint      = "2006.01.02 15:04"
	CSTLayoutYearMonth        = "2006-01"
	CSTLayoutYearMonthChinese = "2006年01月"
	CSTLayoutYearMonthPoint   = "2006.01.02"
	CSTLayoutSecond           = "20060102150405"
	DateDirLayout             = "2006/0101"
	DateDirsLayout            = "2006/01/01"

	DayStartTimeStr = "00:00:00"
	DayEndTimeStr   = "23:59:59"
)

// init 初始化默认的上海时区配置。
func init() {
	var err error
	if ShangHaiTimeLocation, err = time.LoadLocation("Asia/Shanghai"); err != nil {
		panic(err)
	}
	time.Local = ShangHaiTimeLocation
}

// ========== 基础时间构造 ==========

func buildDateOnlyTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, ShangHaiTimeLocation)
}

func buildTimeOnlyTime(hour, minute, second, nanosecond int) time.Time {
	return time.Date(1970, 1, 1, hour, minute, second, nanosecond, ShangHaiTimeLocation)
}

func buildTimeHMTime(hour, minute int) time.Time {
	return buildTimeOnlyTime(hour, minute, 0, 0)
}

func buildClockDuration(hours, minutes, seconds int) time.Duration {
	return time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second
}

// ========== 归一化与解析 ==========

func normalizeToShanghai(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	return value.In(ShangHaiTimeLocation)
}

func normalizeDateOnlyValue(value time.Time) time.Time {
	value = normalizeToShanghai(value)
	if value.IsZero() {
		return time.Time{}
	}
	return buildDateOnlyTime(value.Year(), value.Month(), value.Day())
}

func normalizeTimeOnlyValue(value time.Time) time.Time {
	value = normalizeToShanghai(value)
	if value.IsZero() {
		return time.Time{}
	}
	return buildTimeOnlyTime(value.Hour(), value.Minute(), value.Second(), value.Nanosecond())
}

func normalizeTimeHMValue(value time.Time) time.Time {
	value = normalizeToShanghai(value)
	if value.IsZero() {
		return time.Time{}
	}
	return buildTimeHMTime(value.Hour(), value.Minute())
}

func formatTimeValue(value time.Time, layout string) string {
	value = normalizeToShanghai(value)
	if value.IsZero() {
		return ""
	}
	return value.Format(layout)
}

func parseStringValue(
	value string,
	layout string,
	normalize timeNormalizer,
) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("时间不能为空")
	}
	parsed, err := time.ParseInLocation(layout, value, ShangHaiTimeLocation)
	if err != nil {
		return time.Time{}, errors.New("时间格式错误")
	}
	return normalize(parsed), nil
}

func parseDateTimeString(value string) (time.Time, error) {
	return parseStringValue(
		value,
		CSTLayout,
		normalizeToShanghai,
	)
}

func parseDateOnlyString(value string) (time.Time, error) {
	return parseStringValue(
		value,
		CSTLayoutDate,
		normalizeDateOnlyValue,
	)
}

func parseTimeOnlyString(value string) (time.Time, error) {
	return parseStringValue(
		value,
		CSTLayoutTime,
		normalizeTimeOnlyValue,
	)
}

func parseTimeHMString(value string) (time.Time, error) {
	return parseStringValue(
		value,
		CSTLayoutTimeHM,
		normalizeTimeHMValue,
	)
}

func newParsedTimeValue[T customTimeType](value string, parse stringTimeParser, convert func(time.Time) T) (*T, error) {
	parsed, err := parse(value)
	if err != nil {
		return nil, err
	}
	result := convert(parsed)
	return &result, nil
}

func isZeroCustomTime[T customTimeType](value T) bool {
	return time.Time(value).IsZero()
}

func describeRelativeDateValue[T timeValuer](value T) string {
	return DescribeRelativeDate(value.Time())
}

func addDateToCustomTime[T timeValuer](value T, years, months, days int, convert func(time.Time) T) T {
	return convert(value.Time().AddDate(years, months, days))
}

func addDurationToCustomTime[T timeValuer](value T, delta time.Duration, convert func(time.Time) T) T {
	return convert(value.Time().Add(delta))
}

func beforeTimeValue[T timeValuer](left, right T) bool {
	return left.Time().Before(right.Time())
}

func afterTimeValue[T timeValuer](left, right T) bool {
	return left.Time().After(right.Time())
}

func equalTimeValue[T timeValuer](left, right T) bool {
	return left.Time().Equal(right.Time())
}

func subTimeValue[T timeValuer](left, right T) time.Duration {
	return left.Time().Sub(right.Time())
}

// ========== 当前时间快捷方法 ==========

// Now 返回当前上海时区的时间。
func Now() time.Time {
	return time.Now().In(ShangHaiTimeLocation)
}

// NowPointer 返回当前时间的指针副本，便于与可选参数兼容。
func NowPointer() *time.Time {
	return new(Now())
}

// NowDateTimeString 以标准格式返回当前日期时间字符串。
func NowDateTimeString() string {
	return FormatDateTime(Now())
}

// NowDateString 返回当前日期字符串（YYYY-MM-DD）。
func NowDateString() string {
	return Now().Format(CSTLayoutDate)
}

// NowTimeString 返回当前时间字符串（HH:MM:SS）。
func NowTimeString() string {
	return Now().Format(CSTLayoutTime)
}

// NowAsDateTime 返回当前时间的 DateTime 封装类型。
func NowAsDateTime() DateTime {
	return ToDateTime(Now())
}

// NowAsDateOnly 返回当天零点的 DateOnly 封装类型。
func NowAsDateOnly() DateOnly {
	return ToDateOnly(Now())
}

// NowAsTimeOnly 返回当前时间的 TimeOnly 封装类型。
func NowAsTimeOnly() TimeOnly {
	return ToTimeOnly(Now())
}

// NowAsTimeHM 返回当前时间的时分部分。
func NowAsTimeHM() TimeHM {
	return ToTimeHM(Now())
}

// ToDateTime 将标准 time.Time 转为 DateTime 类型。
func ToDateTime(t time.Time) DateTime {
	return DateTime(normalizeToShanghai(t))
}

// ToDateOnly 将 time.Time 转为 DateOnly，只保留日期部分。
func ToDateOnly(t time.Time) DateOnly {
	return DateOnly(normalizeDateOnlyValue(t))
}

// ToTimeOnly 将 time.Time 转为 TimeOnly，只保留时间部分。
func ToTimeOnly(t time.Time) TimeOnly {
	return TimeOnly(normalizeTimeOnlyValue(t))
}

// ToTimeHM 将 time.Time 转为 TimeHM，只保留时分。
func ToTimeHM(t time.Time) TimeHM {
	return TimeHM(normalizeTimeHMValue(t))
}

// ToTimeOnlyTrimSeconds 将 time.Time 转为无秒的 TimeOnly。
func ToTimeOnlyTrimSeconds(t time.Time) TimeOnly {
	t = normalizeToShanghai(t)
	if t.IsZero() {
		return TimeOnly{}
	}
	return TimeOnly(buildTimeHMTime(t.Hour(), t.Minute()))
}

// ========== DateTime ==========

// NewDateTimeString 用来从字符串解析 DateTime。
func NewDateTimeString(dateString string) (*DateTime, error) {
	return newParsedTimeValue[DateTime](dateString, parseDateTimeString, ToDateTime)
}

// ToDateOnly 用来把完整的 DateTime 取整到日期。
func (dt DateTime) ToDateOnly() DateOnly {
	return ToDateOnly(dt.Time())
}

// ToTimeOnly 用来提取 DateTime 的时分秒部分。
func (dt DateTime) ToTimeOnly() TimeOnly {
	return ToTimeOnly(dt.Time())
}

// ToTimeHM 用来将 DateTime 精确到分钟。
func (dt DateTime) ToTimeHM() TimeHM {
	return ToTimeHM(dt.Time())
}

// Time 用来返回标准库 time.Time 表示。
func (dt DateTime) Time() time.Time {
	return normalizeToShanghai(time.Time(dt))
}

// IsZero 用来判断 DateTime 是否为零值。
func (dt DateTime) IsZero() bool {
	return isZeroCustomTime(dt)
}

// FormatRelativeDate 用来把日期描述为相对时间。
func (dt DateTime) FormatRelativeDate() string {
	return describeRelativeDateValue(dt)
}

// ========== DateOnly ==========

// NewDateOnly 用来创建一个只有日期的值。
func NewDateOnly(year int, month time.Month, day int) DateOnly {
	return ToDateOnly(buildDateOnlyTime(year, month, day))
}

// NewDateOnlyString 用来从字符串解析 DateOnly。
func NewDateOnlyString(dateString string) (*DateOnly, error) {
	return newParsedTimeValue[DateOnly](dateString, parseDateOnlyString, ToDateOnly)
}

// ToDateTime 用来把 DateOnly 转换回 DateTime。
func (d DateOnly) ToDateTime() DateTime {
	return ToDateTime(d.Time())
}

// ToTimeOnly 用来把 DateOnly 视作仅有时间的类型。
func (d DateOnly) ToTimeOnly() TimeOnly {
	return ToTimeOnly(time.Time(d))
}

// ToTimeHM 用来把 DateOnly 转成小时分钟类型。
func (d DateOnly) ToTimeHM() TimeHM {
	return ToTimeHM(time.Time(d))
}

// Time 用来返回标准库 time.Time 表示。
func (d DateOnly) Time() time.Time {
	return normalizeDateOnlyValue(time.Time(d))
}

// IsZero 用来判断 DateOnly 是否为零值。
func (d DateOnly) IsZero() bool {
	return isZeroCustomTime(d)
}

// FormatRelativeDate 用来把日期描述为相对时间。
func (d DateOnly) FormatRelativeDate() string {
	return describeRelativeDateValue(d)
}

// AddDays 用来在日期上增加或减少天数。
func (d DateOnly) AddDays(days int) DateOnly {
	return addDateToCustomTime[DateOnly](d, 0, 0, days, ToDateOnly)
}

// ========== TimeOnly ==========

// NewTimeOnly 用来创建只包含时间部分的值。
func NewTimeOnly(hour, minute, second int) TimeOnly {
	return ToTimeOnly(buildTimeOnlyTime(hour, minute, second, 0))
}

// NewTimeOnlyString 用来从 HH:MM:SS 字符串解析 TimeOnly。
func NewTimeOnlyString(timeString string) (*TimeOnly, error) {
	return newParsedTimeValue[TimeOnly](timeString, parseTimeOnlyString, ToTimeOnly)
}

// ToTimeHM 用来把 TimeOnly 精确到分钟。
func (t TimeOnly) ToTimeHM() TimeHM {
	return ToTimeHM(t.Time())
}

// ToDateOnly 用来把 TimeOnly 当作日期值返回。
func (t TimeOnly) ToDateOnly() DateOnly {
	return ToDateOnly(time.Time(t))
}

// ToDateTime 用来把 TimeOnly 转成 DateTime。
func (t TimeOnly) ToDateTime() DateTime {
	return ToDateTime(t.Time())
}

// Time 用来返回等价的 time.Time 值。
func (t TimeOnly) Time() time.Time {
	return normalizeTimeOnlyValue(time.Time(t))
}

// IsZero 用来判断时间值是否为零。
func (t TimeOnly) IsZero() bool {
	return isZeroCustomTime(t)
}

// AddTime 用来在时间上增加指定的时分秒。
func (t TimeOnly) AddTime(hours, minutes, seconds int) TimeOnly {
	return addDurationToCustomTime[TimeOnly](t,
		buildClockDuration(hours, minutes, seconds),
		ToTimeOnly,
	)
}

// Before 用来判断当前时间是否早于另一个时间。
func (t TimeOnly) Before(other TimeOnly) bool {
	return beforeTimeValue(t, other)
}

// After 用来判断当前时间是否晚于另一个时间。
func (t TimeOnly) After(other TimeOnly) bool {
	return afterTimeValue(t, other)
}

// Equal 用来比较两个时间是否相同。
func (t TimeOnly) Equal(other TimeOnly) bool {
	return equalTimeValue(t, other)
}

// Sub 用来计算两个时间的时间差。
func (t TimeOnly) Sub(other TimeOnly) time.Duration {
	return subTimeValue(t, other)
}

// ========== TimeHM ==========

// NewTimeHM 用来创建只包含小时和分钟的时间。
func NewTimeHM(hour, minute int) TimeHM {
	return ToTimeHM(buildTimeHMTime(hour, minute))
}

// NewTimeHMString 用来从 HH:MM 字符串解析 TimeHM。
func NewTimeHMString(timeString string) (*TimeHM, error) {
	return newParsedTimeValue[TimeHM](timeString, parseTimeHMString, ToTimeHM)
}

// ToTimeOnly 用来从 TimeHM 获取包含秒的时间。
func (t TimeHM) ToTimeOnly() TimeOnly {
	return ToTimeOnly(t.Time())
}

// ToDateTime 用来把 TimeHM 转换为 DateTime。
func (t TimeHM) ToDateTime() DateTime {
	return ToDateTime(t.Time())
}

// ToDateOnly 用来把 TimeHM 转换为 DateOnly。
func (t TimeHM) ToDateOnly() DateOnly {
	return ToDateOnly(time.Time(t))
}

// Time 用来返回等价的 time.Time 值。
func (t TimeHM) Time() time.Time {
	return normalizeTimeHMValue(time.Time(t))
}

// IsZero 用来判断时间是否为空值。
func (t TimeHM) IsZero() bool {
	return isZeroCustomTime(t)
}

// AddTime 用来为小时分钟值增加时长。
func (t TimeHM) AddTime(hours, minutes int) TimeHM {
	return addDurationToCustomTime[TimeHM](t,
		buildClockDuration(hours, minutes, 0),
		ToTimeHM,
	)
}

// Before 用来判断当前值是否早于另一个值。
func (t TimeHM) Before(other TimeHM) bool {
	return beforeTimeValue(t, other)
}

// After 用来判断当前值是否晚于另一个值。
func (t TimeHM) After(other TimeHM) bool {
	return afterTimeValue(t, other)
}

// Equal 用来比较两个时刻是否相同。
func (t TimeHM) Equal(other TimeHM) bool {
	return equalTimeValue(t, other)
}

// Sub 用来计算两个小时分钟值之间的差。
func (t TimeHM) Sub(other TimeHM) time.Duration {
	return subTimeValue(t, other)
}

// ========== 通用解析与格式化 ==========

// ParseDateTime 解析标准日期时间字符串为 time.Time。
func ParseDateTime(value string) (time.Time, error) {
	return parseDateTimeString(value)
}

// ParseDateTimeValue 解析字符串并返回 DateTime 类型。
func ParseDateTimeValue(value string) (DateTime, error) {
	parsed, err := ParseDateTime(value)
	if err != nil {
		return DateTime{}, err
	}
	return ToDateTime(parsed), nil
}

// MustParseDateTimeValue 解析字符串为 DateTime，失败返回零值。
func MustParseDateTimeValue(value string) DateTime {
	parsed, err := ParseDateTime(value)
	if err != nil {
		return DateTime{}
	}
	return ToDateTime(parsed)
}

// ParseDate 解析日期字符串为 time.Time。
func ParseDate(value string) (time.Time, error) {
	return parseDateOnlyString(value)
}

// MustParseDate 解析日期字符串，失败返回零值。
func MustParseDate(value string) time.Time {
	parsed, err := ParseDate(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// ParseDateOnly 解析日期字符串为 DateOnly。
func ParseDateOnly(value string) (DateOnly, error) {
	parsed, err := parseDateOnlyString(value)
	if err != nil {
		return DateOnly{}, err
	}
	return ToDateOnly(parsed), nil
}

// ParseTimeClock 解析时间字符串为 time.Time。
func ParseTimeClock(value string) (time.Time, error) {
	return parseTimeOnlyString(value)
}
func ParseTimeHMClock(value string) (time.Time, error) {
	return parseTimeHMString(value)
}

// MustParseClock 解析时间字符串，失败返回零值。
func MustParseClock(value string) time.Time {
	parsed, err := ParseTimeClock(value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// ParseTimeOnly 解析时间字符串为 TimeOnly。
func ParseTimeOnly(value string) (TimeOnly, error) {
	parsed, err := parseTimeOnlyString(value)
	if err != nil {
		return TimeOnly{}, err
	}
	return ToTimeOnly(parsed), nil
}

// ParseTimeHM 解析时间字符串并返回时分结构。
func ParseTimeHM(value string) (TimeHM, error) {
	parsed, err := parseTimeHMString(value)
	if err != nil {
		return TimeHM{}, err
	}
	return ToTimeHM(parsed), nil
}

// ParseDateAndTimePointer 将日期与时间字符串组合为可选的 time.Time 指针。
func ParseDateAndTimePointer(date string, hourMinuteSecond string) (*time.Time, error) {
	if date == "" {
		return nil, nil
	}
	if hourMinuteSecond != "" {
		date = fmt.Sprintf("%s %s", date, hourMinuteSecond)
	}
	parsed, err := ParseDateTime(date)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

// FormatDateTime 以标准格式输出 time.Time。
func FormatDateTime(t time.Time) string {
	return formatTimeValue(t, CSTLayout)
}

// ParseFuzzyTime 使用 dateparse 模糊解析时间。
func ParseFuzzyTime(value string) (time.Time, error) {
	return dateparse.ParseIn(value, ShangHaiTimeLocation)
}

// FormatDateTimePointer 安全地格式化 time.Time 指针。
func FormatDateTimePointer(t *time.Time) string {
	if t == nil {
		return ""
	}
	return FormatDateTime(*t)
}

// FormatDatePointer 将 time.Time 指针格式化为日期字符串。
func FormatDatePointer(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTimeValue(normalizeDateOnlyValue(*t), CSTLayoutDate)
}

// ParseDateTimePointer 解析日期时间字符串并返回指针。
func ParseDateTimePointer(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := ParseDateTime(value)
	if err != nil {
		return nil
	}
	return new(parsed)
}

// ParseRFC3339Pointer 解析 RFC3339 时间并返回指针。
func ParseRFC3339Pointer(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation(time.RFC3339, value, ShangHaiTimeLocation)
	if err != nil {
		return nil
	}
	return new(parsed)
}

// SecondsToDuration 将秒数转换为 duration。
func SecondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

// UnixFromTime 返回时间的 Unix 秒。
func UnixFromTime(t time.Time) int {
	return int(t.Unix())
}

// FormatUnixDateTime 将 Unix 秒格式化为日期字符串。
func FormatUnixDateTime(unix int64) string {
	return formatTimeValue(time.Unix(unix, 0), CSTLayout)
}

// NowUnix 返回当前时间的 Unix 秒。
func NowUnix() int {
	return UnixFromTime(Now())
}

// NowDateDirectory 生成按日期分层的目录名。
func NowDateDirectory() string {
	return Now().Format(DateDirLayout)
}

// NowAddMinutes 返回当前时间增加指定分钟后的时间。
func NowAddMinutes(minutes int) time.Time {
	return Now().Add(time.Duration(minutes) * time.Minute)
}

// NowSubMinutes 返回当前时间减少指定分钟后的时间。
func NowSubMinutes(minutes int) time.Time {
	return Now().Add(-time.Duration(minutes) * time.Minute)
}

// IsAfterMinutesAgo 判断时间是否晚于若干分钟前。
func IsAfterMinutesAgo(t time.Time, minutes int) bool {
	return t.After(Now().Add(-time.Duration(minutes) * time.Minute))
}

// NowSubHours 返回当前时间减少指定小时后的时间。
func NowSubHours(hours int) time.Time {
	return Now().Add(-time.Duration(hours) * time.Hour)
}

// ========== 描述与范围工具 ==========

// DescribeRelativeDate 返回日期相对于今天的中文描述。
func DescribeRelativeDate(input time.Time) string {
	now := Now()
	y, m, d := input.Date()
	switch {
	case y == now.Year() && m == now.Month() && d == now.Day():
		return "今天"
	case y == now.Year() && m == now.Month() && d == now.AddDate(0, 0, 1).Day():
		return "明天"
	case y == now.Year() && m == now.Month() && d == now.AddDate(0, 0, -1).Day():
		return "昨天"
	case y == now.Year() && m == now.Month():
		return "本月"
	case y == now.Year() && m == now.AddDate(0, -1, 0).Month():
		return "上月"
	default:
		return ""
	}
}

// DescribeRelativeTimeOfDay 根据小时返回一天中的描述。
func DescribeRelativeTimeOfDay(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 0 && hour < 6:
		return "凌晨"
	case hour >= 6 && hour < 12:
		return "上午"
	case hour == 12:
		return "中午"
	case hour >= 13 && hour < 18:
		return "下午"
	case hour >= 18 && hour < 24:
		return "晚上"
	default:
		return ""
	}
}

// TodayRange 返回今天的起止时间范围。
func TodayRange() (DateTime, DateTime) {
	start := beginningOfDay(Now())
	end := start.AddDate(0, 0, 1)
	return buildRange(start, end)
}

// YesterdayRange 返回昨天的起止时间范围。
func YesterdayRange() (DateTime, DateTime) {
	start := beginningOfDay(Now().AddDate(0, 0, -1))
	end := start.AddDate(0, 0, 1)
	return buildRange(start, end)
}

// LastMonthRange 返回上个月的时间范围。
func LastMonthRange() (DateTime, DateTime) {
	start := beginningOfMonth(Now().AddDate(0, -1, 0))
	end := beginningOfMonth(Now())
	return buildRange(start, end)
}

// CurrentMonthRange 返回本月的时间范围。
func CurrentMonthRange() (DateTime, DateTime) {
	start := beginningOfMonth(Now())
	end := beginningOfMonth(Now().AddDate(0, 1, 0))
	return buildRange(start, end)
}

// NextMonthRange 返回下个月的时间范围。
func NextMonthRange() (DateTime, DateTime) {
	start := beginningOfMonth(Now().AddDate(0, 1, 0))
	end := beginningOfMonth(Now().AddDate(0, 2, 0))
	return buildRange(start, end)
}

// LastYearRange 返回上一年的时间范围。
func LastYearRange() (DateTime, DateTime) {
	start := beginningOfYear(Now().AddDate(-1, 0, 0))
	end := beginningOfYear(Now())
	return buildRange(start, end)
}

// CurrentYearRange 返回今年的时间范围。
func CurrentYearRange() (DateTime, DateTime) {
	start := beginningOfYear(Now())
	end := beginningOfYear(Now().AddDate(1, 0, 0))
	return buildRange(start, end)
}

// NextYearRange 返回明年的时间范围。
func NextYearRange() (DateTime, DateTime) {
	start := beginningOfYear(Now().AddDate(1, 0, 0))
	end := beginningOfYear(Now().AddDate(2, 0, 0))
	return buildRange(start, end)
}

// ChineseWeekday 返回中文星期表示。
func ChineseWeekday(t time.Time) string {
	weekdays := []string{"日", "一", "二", "三", "四", "五", "六"}
	return "星期" + weekdays[t.Weekday()]
}

// EnglishWeekday 返回英文星期字符串。
func EnglishWeekday(t time.Time) string {
	return t.Weekday().String()
}

// IsWeekend 判断是否为周末。
func IsWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

// TimeRange 表示一个具备唯一 ID 的时间段。
type TimeRange struct {
	ID    uint64
	Start time.Time
	End   time.Time
}

// IsValid 检查时间段是否有效
func (tr TimeRange) IsValid() bool {
	return !tr.Start.After(tr.End)
}

// HasConflictWith 检查当前时间段是否与另一个时间段冲突
func (tr TimeRange) HasConflictWith(other TimeRange) bool {
	if !tr.IsValid() || !other.IsValid() {
		return false
	}
	return tr.Start.Before(other.End) && tr.End.After(other.Start)
}

// HasTimeConflict 检查多个时间段之间是否有冲突
// 参数: 可变数量的TimeRange，每个TimeRange包含开始时间和结束时间
// 返回 true 表示存在冲突，false 表示无冲突
func HasTimeConflict(timeRanges ...TimeRange) bool {
	// 如果时间段数量少于2个，不可能有冲突
	if len(timeRanges) < 2 {
		return false
	}

	// 检查每一对时间段是否冲突
	for i := range len(timeRanges) {
		for j := i + 1; j < len(timeRanges); j++ {
			if timeRanges[i].HasConflictWith(timeRanges[j]) {
				return true
			}
		}
	}

	return false
}

func HasTimeConflictReturnIDS(ranges ...TimeRange) []uint64 {
	overlappingIDs := make(map[uint64]bool)

	for i := range len(ranges) {
		for j := i + 1; j < len(ranges); j++ {
			// 检查时间重合条件
			if ranges[i].Start.Before(ranges[j].End) && ranges[i].End.After(ranges[j].Start) {
				overlappingIDs[ranges[i].ID] = true
				overlappingIDs[ranges[j].ID] = true
			}
		}
	}

	// 转换为切片
	var result []uint64
	for id := range overlappingIDs {
		result = append(result, id)
	}

	return result
}

func beginningOfDay(t time.Time) time.Time {
	t = normalizeToShanghai(t)
	if t.IsZero() {
		return time.Time{}
	}
	return buildDateOnlyTime(t.Year(), t.Month(), t.Day())
}

func beginningOfMonth(t time.Time) time.Time {
	t = normalizeToShanghai(t)
	if t.IsZero() {
		return time.Time{}
	}
	return buildDateOnlyTime(t.Year(), t.Month(), 1)
}

func beginningOfYear(t time.Time) time.Time {
	t = normalizeToShanghai(t)
	if t.IsZero() {
		return time.Time{}
	}
	return buildDateOnlyTime(t.Year(), 1, 1)
}

func buildRange(start, end time.Time) (DateTime, DateTime) {
	return ToDateTime(start), ToDateTime(end)
}

// GetDay 获取指定时间t的天数
func GetDay(t time.Time) int {
	y, m, d := t.Date()
	dayOfYear := time.Date(y, m, d, 0, 0, 0, 0, ShangHaiTimeLocation).YearDay()
	return y*365 + y/4 - y/100 + y/400 + dayOfYear
}
