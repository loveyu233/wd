package wd

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMonthDayJSONAndParam(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var value MonthDay
		if err := json.Unmarshal([]byte(`"04-17"`), &value); err != nil {
			t.Fatalf("反序列化 MonthDay 失败: %v", err)
		}
		if got := value.String(); got != "04-17" {
			t.Fatalf("MonthDay 字符串错误: got %q", got)
		}
		if got := value.Time(); got.Year() != monthDayStoredYear || got.Month() != time.April || got.Day() != 17 {
			t.Fatalf("MonthDay 固定年份归一化错误: got %v", got)
		}

		data, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("序列化 MonthDay 失败: %v", err)
		}
		if got := string(data); got != `"04-17"` {
			t.Fatalf("MonthDay JSON 输出错误: got %s", got)
		}
	})

	t.Run("param", func(t *testing.T) {
		var value MonthDay
		if err := value.UnmarshalParam("02-29"); err != nil {
			t.Fatalf("解析闰日 MonthDay 失败: %v", err)
		}
		if got := value.Time(); got.Year() != monthDayStoredYear || got.Month() != time.February || got.Day() != 29 {
			t.Fatalf("MonthDay 参数解析错误: got %v", got)
		}
	})
}

func TestMonthDaySQLValueAndScan(t *testing.T) {
	value := NewMonthDay(time.April, 17)

	driverValue, err := value.Value()
	if err != nil {
		t.Fatalf("MonthDay Value 失败: %v", err)
	}
	if got := driverValue.(string); got != "2000-04-17" {
		t.Fatalf("MonthDay 数据库存储值错误: got %q", got)
	}

	var fromString MonthDay
	if err := fromString.Scan("2000-04-17"); err != nil {
		t.Fatalf("MonthDay 扫描日期字符串失败: %v", err)
	}
	if got := fromString.String(); got != "04-17" {
		t.Fatalf("MonthDay 扫描日期字符串后输出错误: got %q", got)
	}

	var fromTime MonthDay
	if err := fromTime.Scan(time.Date(2026, time.April, 17, 12, 30, 0, 0, ShangHaiTimeLocation)); err != nil {
		t.Fatalf("MonthDay 扫描 time.Time 失败: %v", err)
	}
	if got := fromTime.Time(); got.Year() != monthDayStoredYear || got.Month() != time.April || got.Day() != 17 {
		t.Fatalf("MonthDay 扫描 time.Time 后归一化错误: got %v", got)
	}
}

func TestMonthDayRejectsFullDateJSON(t *testing.T) {
	var value MonthDay
	if err := json.Unmarshal([]byte(`"2000-04-17"`), &value); err == nil {
		t.Fatal("MonthDay 不应接受完整日期 JSON")
	}
}
