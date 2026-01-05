package wd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/r3labs/diff/v3"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffReturnLogs 对比两个对象并返回差异，返回的值可以用来记录日志
func DiffReturnLogs(a, b any, tagName ...string) (diff.Changelog, error) {
	var tn = "json"
	if len(tagName) > 0 {
		tn = tagName[0]
	}
	return diff.Diff(a, b, diff.TagName(tn))
}

// DiffReturnLogsBytes 对比两个对象并返回差异，返回值为JSON字节数组，如果有错误则返回nil
func DiffReturnLogsBytes(a, b any, tagName ...string) []byte {
	changelog, err := DiffReturnLogs(a, b, tagName...)
	if err != nil {
		return nil
	}
	marshal, err := json.Marshal(changelog)
	if err != nil {
		return nil
	}
	return marshal
}

// DiffReturnSemanticLogs 在 DiffReturnLogs 的基础上进一步返回语义化描述，便于记录中文日志
// alias 用来指定字段或路径的人性化别名，key 可以是单个字段名，也可以是 "a.b.c" 这种完整路径
func DiffReturnSemanticLogs(a, b any, alias map[string]string, tagName ...string) ([]string, error) {
	changelog, err := DiffReturnLogs(a, b, tagName...)
	if err != nil {
		return nil, err
	}
	return DiffLogsToMessages(changelog, alias), nil
}

// DiffLogsToMessages 将 diff.Changelog 结果转为可读的中文描述
// alias 用来指定字段或路径别名，如 map[string]string{"password":"密码", "car":"汽车"}
func DiffLogsToMessages(changelog diff.Changelog, alias map[string]string) []string {
	msgs := make([]string, 0, len(changelog))
	for _, change := range changelog {
		if msg := formatDiffChange(change, alias); msg != "" {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// DiffText 对比两个文本并返回差异，可以用来对比字符串
func DiffText(text1, text2 string, checkLines ...bool) []diffmatchpatch.Diff {
	var checkLine = false
	if len(checkLines) > 0 {
		checkLine = checkLines[0]
	}
	return diffmatchpatch.New().DiffMain(text1, text2, checkLine)
}

// DiffReturnHtml 对比两个文本并返回差异，返回结果是一个html字符串，可以在前端页面直接展示
func DiffReturnHtml(text1, text2 string, checkLines ...bool) string {
	var checkLine = false
	if len(checkLines) > 0 {
		checkLine = checkLines[0]
	}
	dmp := diffmatchpatch.New()
	return dmp.DiffPrettyHtml(dmp.DiffMain(text1, text2, checkLine))
}

// DiffReturnColorText 对比两个文本并返回差异，返回结果是一个带颜色标识的文本，主要用于终端显示
func DiffReturnColorText(text1, text2 string, checkLines ...bool) string {
	var checkLine = false
	if len(checkLines) > 0 {
		checkLine = checkLines[0]
	}
	dmp := diffmatchpatch.New()
	return dmp.DiffPrettyText(dmp.DiffMain(text1, text2, checkLine))
}

func formatDiffChange(change diff.Change, alias map[string]string) string {
	pathDesc := formatDiffPath(change.Path, alias)
	switch strings.ToLower(change.Type) {
	case "create":
		return fmt.Sprintf("新增了 %s 值为【%s】", pathDesc, formatDiffValue(change.To))
	case "delete":
		return fmt.Sprintf("删除了 %s 原值【%s】", pathDesc, formatDiffValue(change.From))
	case "update":
		fallthrough
	default:
		return fmt.Sprintf("修改了 %s 从【%s】修改为【%s】", pathDesc, formatDiffValue(change.From), formatDiffValue(change.To))
	}
}

func formatDiffPath(path []string, alias map[string]string) string {
	if len(path) == 0 {
		return "对象"
	}

	if alias != nil {
		if v, ok := alias[strings.Join(path, ".")]; ok && v != "" {
			return v
		}
	}

	parts := make([]string, 0, len(path))
	for _, segment := range path {
		name := segment
		if alias != nil {
			if v, ok := alias[segment]; ok && v != "" {
				name = v
			}
		}

		if name == segment {
			if idx, err := strconv.Atoi(segment); err == nil {
				name = formatOrdinal(idx)
			}
		}

		parts = append(parts, name)
	}

	return strings.Join(parts, "的")
}

func formatDiffValue(val interface{}) string {
	if val == nil {
		return "空"
	}

	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case fmt.GoStringer:
		return v.GoString()
	}

	if b, err := json.Marshal(val); err == nil {
		return string(b)
	}

	return fmt.Sprint(val)
}

func formatOrdinal(idx int) string {
	ordinal := idx + 1
	if ordinal <= 0 {
		return fmt.Sprintf("第%d个值", ordinal)
	}

	var label string
	if ordinal < 100 {
		label = numberToChinese(ordinal)
	} else {
		label = strconv.Itoa(ordinal)
	}

	return fmt.Sprintf("第%s个值", label)
}

func numberToChinese(n int) string {
	digits := []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	if n <= 0 {
		return strconv.Itoa(n)
	}
	if n < 10 {
		return digits[n]
	}
	if n == 10 {
		return "十"
	}
	if n < 20 {
		return "十" + digits[n-10]
	}
	if n < 100 {
		tens := n / 10
		units := n % 10
		if units == 0 {
			return digits[tens] + "十"
		}
		return digits[tens] + "十" + digits[units]
	}
	return strconv.Itoa(n)
}
