package wd

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// GetPositionChars n为正数则从前开始 负数则从后开始
func GetPositionChars(str string, n int) string {
	runes := []rune(str)
	length := len(runes)

	if n <= 0 {
		n += length
		if n >= length || n < 0 {
			return str
		}
		return string(runes[n:])
	}

	if n >= length {
		return str
	}

	return string(runes[:n])
}

// ConvertStringToUint32 用来校验并将数字字符串转换成 uint32。
func ConvertStringToUint32(str string) (uint32, error) {
	// 去除空格
	str = strings.TrimSpace(str)

	// 检查空字符串
	if str == "" {
		return 0, fmt.Errorf("输入字符串为空")
	}

	var result uint64
	for i := range len(str) {
		char := str[i]
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("字符串包含非数字字符: %s", str)
		}
		result = result*10 + uint64(char-'0')
		if result > uint64(^uint32(0)) {
			return 0, fmt.Errorf("转换失败: 数值超出 uint32 范围")
		}
	}

	return uint32(result), nil
}

// ConvertStringToUint32Simple 用来在失败时返回 0 的便捷转换。
func ConvertStringToUint32Simple(str string) uint32 {
	result, err := ConvertStringToUint32(str)
	if err != nil {
		return 0
	}
	return result
}

// GetGenderFromIDCard 用来根据身份证号码推断性别。
func GetGenderFromIDCard(idcard string) string {
	idcard = normalizeChineseIDCard(idcard)
	if !isValidNormalizedChineseIDCard(idcard) {
		return "未知"
	}
	digit := idcard[16] - '0'

	// 判断奇偶性
	if digit%2 == 0 {
		return "女"
	}
	return "男"
}

// ValidateChineseMobile 用来校验中国大陆手机号格式。
func ValidateChineseMobile(mobile string) bool {
	return isValidNormalizedChineseMobile(normalizeChineseMobile(mobile))
}

// MaskMobileCustom 用来按自定义规则脱敏手机号。
func MaskMobileCustom(mobile string, prefixLen, suffixLen int, maskChar rune) string {
	mobile = normalizeChineseMobile(mobile)

	// 如果不是有效的手机号格式，返回原字符串
	if !isValidNormalizedChineseMobile(mobile) {
		return mobile
	}

	// 验证参数有效性
	if prefixLen < 0 || suffixLen < 0 || prefixLen+suffixLen >= len(mobile) {
		return mobile
	}

	// 计算中间需要遮蔽的位数
	maskLen := len(mobile) - prefixLen - suffixLen

	// 构建遮蔽字符串
	maskStr := strings.Repeat(string(maskChar), maskLen)

	// 返回脱敏后的手机号
	return mobile[:prefixLen] + maskStr + mobile[len(mobile)-suffixLen:]
}

// ValidateChineseIDCard 用来校验身份证号是否合法。
func ValidateChineseIDCard(idCard string) bool {
	return isValidNormalizedChineseIDCard(normalizeChineseIDCard(idCard))
}

// MaskMobile 用来以默认规则对手机号进行脱敏。
func MaskMobile(mobile string) string {
	mobile = normalizeChineseMobile(mobile)

	// 如果不是有效的手机号格式，返回原字符串
	if !isValidNormalizedChineseMobile(mobile) {
		return mobile
	}

	// 对手机号进行脱敏：前3位 + 5个* + 后4位
	return mobile[:3] + "****" + mobile[7:]
}

// MaskIDCardCustom 用来自定义身份证号码的脱敏方案。
func MaskIDCardCustom(idCard string, prefixLen, suffixLen int, maskChar rune) string {
	idCard = normalizeChineseIDCard(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !isValidNormalizedChineseIDCard(idCard) {
		return idCard
	}

	// 验证参数有效性
	if prefixLen < 0 || suffixLen < 0 || prefixLen+suffixLen >= len(idCard) {
		return idCard
	}

	// 计算中间需要遮蔽的位数
	maskLen := len(idCard) - prefixLen - suffixLen

	// 构建遮蔽字符串
	maskStr := strings.Repeat(string(maskChar), maskLen)

	// 返回脱敏后的身份证号
	return idCard[:prefixLen] + maskStr + idCard[len(idCard)-suffixLen:]
}

// MaskIDCardBirthday 用来隐藏身份证中的生日与顺序码。
func MaskIDCardBirthday(idCard string) string {
	idCard = normalizeChineseIDCard(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !isValidNormalizedChineseIDCard(idCard) {
		return idCard
	}

	// 身份证结构：前6位地区码 + 8位生日 + 3位顺序码 + 1位校验码
	// 保留地区码和校验码，隐藏生日和顺序码
	return idCard[:6] + "***********" + idCard[17:]
}

// MaskIDCard 用来以固定规则遮蔽身份证号。
func MaskIDCard(idCard string) string {
	idCard = normalizeChineseIDCard(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !isValidNormalizedChineseIDCard(idCard) {
		return idCard
	}

	// 对身份证号进行脱敏：前6位 + 8个* + 后4位
	return idCard[:6] + "********" + idCard[14:]
}

// MaskUsername 用来只保留用户名第一个字符并遮蔽剩余部分。
func MaskUsername(username string, maskLastName ...bool) string {
	if utf8.RuneCountInString(username) <= 2 {
		if len(maskLastName) > 0 && maskLastName[0] {
			return fmt.Sprintf("*%s", GetPositionChars(username, -1))
		}
		return fmt.Sprintf("%s*", GetPositionChars(username, 1))
	}

	return fmt.Sprintf("%s*%s", GetPositionChars(username, 1), GetPositionChars(username, -1))
}

// ReplacePathParamsFast 处理路径参数替换为*
func ReplacePathParamsFast(path string) string {
	if path == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(path)) // 预分配空间，提高性能

	inParam := false

	for i := 0; i < len(path); i++ {
		ch := path[i]

		if ch == ':' {
			// 检查是否是路径参数开头
			if i == 0 || path[i-1] == '/' {
				result.WriteByte('*')
				inParam = true
				// 跳过参数名部分
				for i < len(path) && path[i] != '/' {
					i++
				}
				if i < len(path) {
					result.WriteByte(path[i])
				}
				continue
			}
		}

		if !inParam {
			result.WriteByte(ch)
		}

		if ch == '/' {
			inParam = false
		}
	}

	return result.String()
}

func normalizeChineseMobile(mobile string) string {
	needsNormalize := false
	for i := range len(mobile) {
		switch mobile[i] {
		case ' ', '-':
			needsNormalize = true
		}
		if needsNormalize {
			break
		}
	}
	if !needsNormalize {
		return mobile
	}

	var builder strings.Builder
	builder.Grow(len(mobile))
	for i := range len(mobile) {
		switch mobile[i] {
		case ' ', '-':
			continue
		default:
			builder.WriteByte(mobile[i])
		}
	}
	return builder.String()
}

func isValidNormalizedChineseMobile(mobile string) bool {
	if len(mobile) != 11 {
		return false
	}
	if mobile[0] != '1' || mobile[1] < '3' || mobile[1] > '9' {
		return false
	}
	for i := 2; i < len(mobile); i++ {
		if mobile[i] < '0' || mobile[i] > '9' {
			return false
		}
	}
	return true
}

func normalizeChineseIDCard(idCard string) string {
	needsNormalize := false
	for i := range len(idCard) {
		ch := idCard[i]
		if ch == ' ' || ('a' <= ch && ch <= 'z') {
			needsNormalize = true
			break
		}
	}
	if !needsNormalize {
		return idCard
	}

	var builder strings.Builder
	builder.Grow(len(idCard))
	for i := range len(idCard) {
		ch := idCard[i]
		if ch == ' ' {
			continue
		}
		if 'a' <= ch && ch <= 'z' {
			ch = ch - 'a' + 'A'
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func isValidNormalizedChineseIDCard(idCard string) bool {
	if len(idCard) != 18 {
		return false
	}
	for i := range 17 {
		if idCard[i] < '0' || idCard[i] > '9' {
			return false
		}
	}
	lastChar := idCard[17]
	if lastChar != 'X' && (lastChar < '0' || lastChar > '9') {
		return false
	}

	weights := [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := [...]byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := range 17 {
		sum += int(idCard[i]-'0') * weights[i]
	}
	return lastChar == checkCodes[sum%11]
}
