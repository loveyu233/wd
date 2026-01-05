package wd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// GetLastNChars 用来返回字符串结尾的 n 个字符。
func GetLastNChars(str string, n int) string {
	runes := []rune(str)
	length := len(runes)

	if n <= 0 {
		return ""
	}

	if n >= length {
		return str
	}

	return string(runes[length-n:])
}

// GetFirstNChars 用来返回字符串开头的 n 个字符。
func GetFirstNChars(str string, n int) string {
	runes := []rune(str)
	length := len(runes)

	if n <= 0 {
		return ""
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

	// 检查是否全为数字
	for _, char := range str {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("字符串包含非数字字符: %s", str)
		}
	}

	// 使用strconv.ParseUint自动处理前导零
	// ParseUint会自动去除前导零并转换为数字
	result, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("转换失败: %v", err)
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

// GetGenderFormIDCard 用来根据身份证号码推断性别。
func GetGenderFormIDCard(idcard string) string {
	if !ValidateChineseIDCard(idcard) {
		return "未知"
	}
	// 获取第17位数字(索引为16)
	// 中国身份证第17位数字表示性别：奇数为男性，偶数为女性
	genderDigit := idcard[16:17]

	// 将字符转换为数字
	digit, err := strconv.Atoi(genderDigit)
	if err != nil {
		return "无效身份证号"
	}

	// 判断奇偶性
	if digit%2 == 0 {
		return "女"
	}
	return "男"
}

// ValidateChineseMobile 用来校验中国大陆手机号格式。
func ValidateChineseMobile(mobile string) bool {
	// 去除空格和特殊字符
	mobile = strings.ReplaceAll(mobile, " ", "")
	mobile = strings.ReplaceAll(mobile, "-", "")

	// 中国手机号正则表达式
	// 1开头，第二位是3-9，总共11位数字
	pattern := `^1[3-9]\d{9}$`
	matched, _ := regexp.MatchString(pattern, mobile)
	return matched
}

// MaskMobileCustom 用来按自定义规则脱敏手机号。
func MaskMobileCustom(mobile string, prefixLen, suffixLen int, maskChar rune) string {
	// 去除空格和特殊字符
	mobile = strings.ReplaceAll(mobile, " ", "")
	mobile = strings.ReplaceAll(mobile, "-", "")

	// 如果不是有效的手机号格式，返回原字符串
	if !ValidateChineseMobile(mobile) {
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
	// 去除空格
	idCard = strings.ReplaceAll(idCard, " ", "")
	idCard = strings.ToUpper(idCard)

	// 检查长度，必须是18位
	if len(idCard) != 18 {
		return false
	}

	// 检查前17位是否都是数字
	for i := 0; i < 17; i++ {
		if idCard[i] < '0' || idCard[i] > '9' {
			return false
		}
	}

	// 检查最后一位（校验码）
	lastChar := idCard[17]
	if lastChar != 'X' && (lastChar < '0' || lastChar > '9') {
		return false
	}

	// 计算校验码
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

	sum := 0
	for i := 0; i < 17; i++ {
		digit, _ := strconv.Atoi(string(idCard[i]))
		sum += digit * weights[i]
	}

	expectedCheckCode := checkCodes[sum%11]
	return byte(lastChar) == expectedCheckCode
}

// MaskMobile 用来以默认规则对手机号进行脱敏。
func MaskMobile(mobile string) string {
	// 去除空格和特殊字符
	mobile = strings.ReplaceAll(mobile, " ", "")
	mobile = strings.ReplaceAll(mobile, "-", "")

	// 如果不是有效的手机号格式，返回原字符串
	if !ValidateChineseMobile(mobile) {
		return mobile
	}

	// 对手机号进行脱敏：前3位 + 5个* + 后4位
	return mobile[:3] + "****" + mobile[7:]
}

// MaskIDCardCustom 用来自定义身份证号码的脱敏方案。
func MaskIDCardCustom(idCard string, prefixLen, suffixLen int, maskChar rune) string {
	// 去除空格
	idCard = strings.ReplaceAll(idCard, " ", "")
	idCard = strings.ToUpper(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !ValidateChineseIDCard(idCard) {
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
	// 去除空格
	idCard = strings.ReplaceAll(idCard, " ", "")
	idCard = strings.ToUpper(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !ValidateChineseIDCard(idCard) {
		return idCard
	}

	// 身份证结构：前6位地区码 + 8位生日 + 3位顺序码 + 1位校验码
	// 保留地区码和校验码，隐藏生日和顺序码
	return idCard[:6] + "***********" + idCard[17:]
}

// MaskIDCard 用来以固定规则遮蔽身份证号。
func MaskIDCard(idCard string) string {
	// 去除空格
	idCard = strings.ReplaceAll(idCard, " ", "")
	idCard = strings.ToUpper(idCard)

	// 如果不是有效的身份证号格式，返回原字符串
	if !ValidateChineseIDCard(idCard) {
		return idCard
	}

	// 对身份证号进行脱敏：前6位 + 8个* + 后4位
	return idCard[:6] + "********" + idCard[14:]
}

// MaskUsername 用来只保留用户名第一个字符并遮蔽剩余部分。
func MaskUsername(username string) string {
	return GetFirstNChars(username, 1) + "*"
}
