package wd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"hash/fnv"
	"math/big"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/xid"
)

// GetUUID 用来生成一个标准的 UUID 字符串。
func GetUUID() string {
	return uuid.NewString()
}

// GetXID 用来生成紧凑的 XID。
func GetXID() string {
	return xid.New().String()
}

const envSnowflakeWorkerID = "WD_SNOWFLAKE_WORKER_ID"

var (
	worker          *Worker
	workerMu        sync.RWMutex
	workerInitOnce  sync.Once
	workerInitError error
)

// InitSnowflakeWorker 用来显式设置雪花算法 worker ID，建议在分布式部署场景下启动时调用。
func InitSnowflakeWorker(workerID int64) error {
	w, err := NewWorker(workerID)
	if err != nil {
		return err
	}

	workerMu.Lock()
	defer workerMu.Unlock()
	worker = w
	return nil
}

// GetSnowflakeID 用来产生分布式雪花 ID。
func GetSnowflakeID() int64 {
	id, err := GetSnowflakeIDErr()
	if err != nil {
		panic(err)
	}
	return id
}

// GetSnowflakeIDErr 用来产生分布式雪花 ID，并把初始化错误返回给调用方。
func GetSnowflakeIDErr() (int64, error) {
	if err := ensureSnowflakeWorker(); err != nil {
		return 0, err
	}

	workerMu.RLock()
	defer workerMu.RUnlock()

	if worker == nil {
		return 0, errors.New("snowflake worker not initialized")
	}
	return worker.GetId(), nil
}

func ensureSnowflakeWorker() error {
	workerMu.RLock()
	if worker != nil {
		workerMu.RUnlock()
		return nil
	}
	workerMu.RUnlock()

	workerInitOnce.Do(func() {
		workerID, err := defaultSnowflakeWorkerID()
		if err != nil {
			workerInitError = err
			return
		}
		workerInitError = InitSnowflakeWorker(workerID)
	})

	if workerInitError != nil {
		return workerInitError
	}

	workerMu.RLock()
	defer workerMu.RUnlock()
	if worker == nil {
		return errors.New("snowflake worker not initialized")
	}
	return nil
}

func defaultSnowflakeWorkerID() (int64, error) {
	if raw := strings.TrimSpace(os.Getenv(envSnowflakeWorkerID)); raw != "" {
		workerID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s 无法解析为整数: %w", envSnowflakeWorkerID, err)
		}
		return workerID, nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		return 0, fmt.Errorf("获取主机名失败: %w", err)
	}

	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hostname))
	return int64(hasher.Sum32() % uint32(workerMax+1)), nil
}

// RandomString 用来生成指定长度的随机字符串。
func RandomString(length int) (string, error) {
	var charset = RandomCharacterSetAllStr().String()
	result := make([]byte, length)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}

// RandomStringNoErr 用来快速生成一个 6 位随机字符串。
func RandomStringNoErr() string {
	var charset = RandomCharacterSetAllStr().String()
	var seededRand = mrand.New(mrand.NewSource(Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandomStringWithPrefix 用来生成带前后缀的随机字符串。
func RandomStringWithPrefix(length int, prefix, suffix string) (string, error) {
	if length <= len(prefix)+len(suffix) {
		return "", errors.New("prefix + suffix <= length")
	}

	randomLength := length - len(prefix) - len(suffix)
	randomPart, err := RandomString(randomLength)
	if err != nil {
		return "", err
	}

	return prefix + randomPart + suffix, nil
}

type RandomCharacterSet string

// String 用来返回字符集的实际内容。
func (r RandomCharacterSet) String() string {
	return string(r)
}

// RandomCharacterSetAllStr 用来返回包含大小写字母和数字的字符集。
func RandomCharacterSetAllStr() RandomCharacterSet {
	return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
}

// RandomCharacterSetLowerStr 用来返回全部小写字母字符集。
func RandomCharacterSetLowerStr() RandomCharacterSet {
	return "abcdefghijklmnopqrstuvwxyz"
}

// RandomCharacterSetLowerStrExcludeCharIO 用来返回去除易混淆字符的全小写字符集。
func RandomCharacterSetLowerStrExcludeCharIO() RandomCharacterSet {
	return "abcdefghjklmnpqrstuvwxyz"
}

// RandomCharacterSetUpperStr 用来返回全部大写字母字符集。
func RandomCharacterSetUpperStr() RandomCharacterSet {
	return "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
}

// RandomCharacterSetUpperStrExcludeCharIO 用来返回去掉 I/O 的大写字符集。
func RandomCharacterSetUpperStrExcludeCharIO() RandomCharacterSet {
	return "ABCDEFGHJKLMNPQRSTUVWXYZ"
}

// RandomCharacterSetNumberStr 用来返回 0-9 的数字字符集。
func RandomCharacterSetNumberStr() RandomCharacterSet {
	return "0123456789"
}

// RandomCharacterSetNumberStrExcludeCharo1 用来返回去除 0/1 的数字字符集。
func RandomCharacterSetNumberStrExcludeCharo1() RandomCharacterSet {
	return "23456789"
}

// RandomCharacterExcludeErrorPronCharacters 用来构建去除易混淆字符的集合。
func RandomCharacterExcludeErrorPronCharacters() RandomCharacterSet {
	return "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjklmnpqrstuvwxyz"
}

// Random 用来从指定字符集中生成随机字符串。
func Random(strLen int64, characterSet ...RandomCharacterSet) string {
	var charset string
	if len(characterSet) == 0 {
		charset = RandomCharacterSetAllStr().String()
	} else {
		for i := range characterSet {
			charset += characterSet[i].String()
		}
	}
	var seededRand = mrand.New(mrand.NewSource(Now().UnixNano()))
	b := make([]byte, strLen)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// RandomExcludeErrorPronCharacters 用来生成没有易混淆字符的随机串。
func RandomExcludeErrorPronCharacters(strLen int64) string {
	return Random(strLen, RandomCharacterExcludeErrorPronCharacters())
}

// RandomIntRange 用来生成 [left, right] 闭区间的随机整数。
func RandomIntRange(left, right int) int {
	if left >= right {
		panic("左必须小于右")
	}
	seededRand := mrand.New(mrand.NewSource(Now().UnixNano()))
	return seededRand.Intn(right-left+1) + left
}
