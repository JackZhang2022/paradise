// Package crypto 提供加密保险库所需的底层密码学操作。
//
// 本包是纯函数包，不产生日志等副作用，所有错误通过返回值传递。
//
// 核心能力：
//   - 密钥派生：Argon2id 从密码+盐值派生 256-bit 密钥
//   - 数据加密：AES-256-GCM 加解密
//   - 名称派生：HMAC-SHA256 派生加密文件名/目录名
//   - 文件加解密：嵌入原始文件名后加密，解密时提取
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	KeySize      = 32
	NonceSize    = 12
	SaltSize     = 32
	ArgonTime    = 3
	ArgonMem     = 64 * 1024
	ArgonThreads = 4
	NameHashSize = 16
)

var (
	ErrDecryptFailed = errors.New("解密失败，密钥可能不正确")
	ErrDataCorrupted = errors.New("文件数据损坏")
)

// DeriveKey 使用 Argon2id 从密码派生密钥
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMem, ArgonThreads, KeySize)
}

// Encrypt 使用 AES-256-GCM 加密数据
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES 密码器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 nonce 失败: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt 使用 AES-256-GCM 解密数据
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES 密码器失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 失败: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptFailed
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptFailed
	}
	return plaintext, nil
}

// DeriveEncryptedName 使用 HMAC-SHA256 派生短加密名称
func DeriveEncryptedName(name string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(name))
	hash := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(hash[:NameHashSize])
}

// EncryptFileWithName 加密文件内容，同时嵌入原始文件名
func EncryptFileWithName(plaintext []byte, originalName string, key []byte) ([]byte, error) {
	nameBytes := []byte(originalName)
	nameLen := uint16(len(nameBytes))

	buf := make([]byte, 2+len(nameBytes)+len(plaintext))
	binary.BigEndian.PutUint16(buf[0:2], nameLen)
	copy(buf[2:], nameBytes)
	copy(buf[2+len(nameBytes):], plaintext)

	return Encrypt(buf, key)
}

// DecryptFileWithName 解密文件内容，提取原始文件名和文件内容
func DecryptFileWithName(ciphertext []byte, key []byte) (originalName string, plaintext []byte, err error) {
	data, err := Decrypt(ciphertext, key)
	if err != nil {
		return "", nil, err
	}
	if len(data) < 2 {
		return "", nil, ErrDataCorrupted
	}
	nameLen := binary.BigEndian.Uint16(data[0:2])
	if int(2+nameLen) > len(data) {
		return "", nil, ErrDataCorrupted
	}
	originalName = string(data[2 : 2+nameLen])
	plaintext = data[2+nameLen:]
	return originalName, plaintext, nil
}

// EncryptDirName 加密目录名，返回加密后的名称和元数据内容
func EncryptDirName(originalName string, key []byte) (encryptedName string, metadata []byte, err error) {
	encryptedName = DeriveEncryptedName(originalName, key)
	metadata, err = Encrypt([]byte(originalName), key)
	return
}

// DecryptDirName 从元数据解密目录名
func DecryptDirName(metadata []byte, key []byte) (string, error) {
	data, err := Decrypt(metadata, key)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
