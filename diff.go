package wd

import (
	"encoding/json"

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
