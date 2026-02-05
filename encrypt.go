package wd

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type EncryptedResponse struct {
	Data      string `json:"data"`      // 加密的数据
	Timestamp int64  `json:"timestamp"` // 时间戳
	Nonce     string `json:"nonce"`     // 随机数，增加安全性
}

// encryptAESGCM 用来使用 AES-GCM 加密明文并返回 Base64 文本。
func encryptAESGCM(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptData 用来序列化数据并返回加密后的响应体。
func EncryptData(data any, custom func(now int64) (key, nonce string)) (*EncryptedResponse, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	key, nonce := custom(now)

	// 加密数据
	encryptedData, err := encryptAESGCM(jsonData, []byte(key))
	if err != nil {
		return nil, err
	}

	return &EncryptedResponse{
		Data:      encryptedData,
		Timestamp: now,
		Nonce:     nonce,
	}, nil
}

// PasswordEncryption 用来对明文密码进行 bcrypt 加密。
func PasswordEncryption(password string) (string, error) {
	fromPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(fromPassword), nil
}

// PasswordCompare 用来对比密码是否正确，inputPassword 输入的密码，originalPassword 原始密码,两个密码均不能为空字符串
func PasswordCompare(inputPassword, originalPassword string) bool {
	if inputPassword == "" || originalPassword == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(inputPassword), []byte(originalPassword)) == nil
}

// PasswordValidateStrength 用来检测密码的长度和复杂度是否合规。
func PasswordValidateStrength(password string, minLen, maxLen int) bool {
	if len(password) < minLen || len(password) > maxLen {
		return false
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, char := range password {
		if char == ' ' { // 如果包含空格，直接返回 false
			return false
		}

		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}
