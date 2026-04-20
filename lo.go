package wd

import (
	"reflect"
)

// LoMap 用来对切片执行映射转换。
func LoMap[T any, R any](collection []T, iteratee func(item T, index int) R) []R {
	if len(collection) == 0 {
		return []R{}
	}
	result := make([]R, len(collection))
	for index, item := range collection {
		result[index] = iteratee(item, index)
	}
	return result
}

// LoSliceToMap 用来把切片转换为键值映射。
func LoSliceToMap[T any, K comparable, V any](collection []T, transform func(item T) (K, V)) map[K]V {
	result := make(map[K]V, len(collection))
	for _, item := range collection {
		key, value := transform(item)
		result[key] = value
	}
	return result
}

// LoTernary 用来模拟条件表达式并返回两个值之一。
func LoTernary[T any](condition bool, ifOutput T, elseOutput T) T {
	if condition {
		return ifOutput
	}
	return elseOutput
}

// LoTernaryFunc 用来在条件成立时延迟执行对应函数。
func LoTernaryFunc[T any](condition bool, ifFunc func() T, elseFunc func() T) T {
	if condition {
		return ifFunc()
	}
	return elseFunc()
}

// LoWithout 用来返回排除了指定元素的新切片。
func LoWithout[T comparable, Slice interface{ ~[]T }](collection Slice, exclude ...T) Slice {
	if len(collection) == 0 || len(exclude) == 0 {
		return append(Slice(nil), collection...)
	}
	excludeSet := make(map[T]struct{}, len(exclude))
	for _, item := range exclude {
		excludeSet[item] = struct{}{}
	}
	result := make(Slice, 0, len(collection))
	for _, item := range collection {
		if _, ok := excludeSet[item]; ok {
			continue
		}
		result = append(result, item)
	}
	return result
}

// LoUniq 用来去除切片中的重复值。
func LoUniq[T comparable, Slice interface{ ~[]T }](collection Slice) Slice {
	if len(collection) == 0 {
		return Slice{}
	}
	seen := make(map[T]struct{}, len(collection))
	result := make(Slice, 0, len(collection))
	for _, item := range collection {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// LoContains 用来判断切片是否包含指定元素。
func LoContains[T comparable](collection []T, element T) bool {
	for _, item := range collection {
		if item == element {
			return true
		}
	}
	return false
}

// IsPtr 用来检查传入对象是否为非空指针。
func IsPtr(target any) bool {
	objValue := reflect.ValueOf(target)
	if objValue.Kind() != reflect.Ptr {
		return false
	}

	if objValue.IsNil() {
		return false
	}

	return true
}

// LoToPtr 用来返回值的指针表示。
func LoToPtr[T any](x T) *T {
	return &x
}

// LoFromPtr 用来在指针为空时返回零值，否则返回其内容。
func LoFromPtr[T any](x *T) T {
	if x == nil {
		var zero T
		return zero
	}
	return *x
}
