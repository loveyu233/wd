package wd

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// DecimalYuanToFen 用来将金额从元转换成以分为单位的整数。
func DecimalYuanToFen(value decimal.Decimal) int64 {
	return value.Mul(decimal.NewFromInt(100)).IntPart()
}

// DecimalFenToYuan 用来把以分表示的金额转换为 decimal 元。
func DecimalFenToYuan(value int64) decimal.Decimal {
	return decimal.NewFromInt(value).Div(decimal.NewFromInt(100))
}

// DecimalFenToYuanStr 用来生成带“元”单位的金额字符串。
func DecimalFenToYuanStr(value int64) string {
	return fmt.Sprintf("%v元", DecimalFenToYuan(value))
}

// DecimalAddsSubsGteZero adds相加后减去subs返回是否大于等于0和相减后的结果值以及添加和减去值
func DecimalAddsSubsGteZero(adds []decimal.Decimal, subs []decimal.Decimal) (gteZero bool, residual decimal.Decimal, addCount decimal.Decimal, subCount decimal.Decimal) {
	for _, add := range adds {
		addCount = addCount.Add(add)
	}
	for _, sub := range subs {
		subCount = subCount.Add(sub)
	}

	residual = addCount.Sub(subCount)
	return residual.GreaterThanOrEqual(decimal.Zero), residual, addCount, subCount
}
