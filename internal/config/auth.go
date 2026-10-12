package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"paradise/internal/crypto"
	"paradise/internal/logger"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordMismatch = errors.New("密码不正确")
	ErrNoPassword       = errors.New("未设置密码，请先注册")
)

// IsFirstRun 检查是否首次运行（配置中无密码哈希即视为首次）
func IsFirstRun() bool {
	cfg, err := LoadConfig()
	if err != nil {
		return true
	}
	return cfg.PasswordHash == ""
}

// SetPassword 首次设置密码，同时生成并持久化盐值
func SetPassword(password string) error {
	if len(password) < 6 {
		logger.Warn("密码设置失败：密码长度不足")
		return errors.New("密码长度至少 6 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("密码哈希失败", "error", err)
		return fmt.Errorf("密码哈希失败: %w", err)
	}

	salt, err := GenerateSalt()
	if err != nil {
		logger.Error("生成盐值失败", "error", err)
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		cfg = &AppConfig{}
	}
	cfg.PasswordHash = string(hash)
	cfg.Salt = EncodeSalt(salt)

	if err := SaveConfig(cfg); err != nil {
		logger.Error("保存配置失败", "error", err)
		return err
	}
	logger.Info("密码设置成功")
	return nil
}

// VerifyPassword 验证密码
func VerifyPassword(password string) error {
	cfg, err := LoadConfig()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		return err
	}
	if cfg.PasswordHash == "" {
		logger.Warn("未设置密码，首次运行")
		return ErrNoPassword
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(password)); err != nil {
		logger.Warn("密码验证失败")
		return ErrPasswordMismatch
	}
	logger.Info("密码验证成功")
	return nil
}

// LoadSalt 从配置中加载持久化盐值
func LoadSalt() ([]byte, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	if cfg.Salt == "" {
		return nil, errors.New("盐值未设置，请先初始化")
	}
	return DecodeSalt(cfg.Salt)
}

// ChangePassword 修改密码：验证旧密码，更新哈希，返回旧密钥和新密钥
func ChangePassword(oldPassword, newPassword string) (oldKey, newKey []byte, err error) {
	if len(newPassword) < 6 {
		logger.Warn("修改密码失败：新密码长度不足")
		return nil, nil, errors.New("新密码长度至少 6 位")
	}

	if err := VerifyPassword(oldPassword); err != nil {
		logger.Warn("修改密码失败：旧密码验证失败", "error", err)
		return nil, nil, fmt.Errorf("旧密码验证失败: %w", err)
	}

	salt, err := LoadSalt()
	if err != nil {
		logger.Error("修改密码失败：加载盐值失败", "error", err)
		return nil, nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error("新密码哈希失败", "error", err)
		return nil, nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		logger.Error("加载配置失败", "error", err)
		return nil, nil, err
	}
	cfg.PasswordHash = string(hash)
	if err := SaveConfig(cfg); err != nil {
		logger.Error("保存配置失败", "error", err)
		return nil, nil, err
	}

	oldKey = crypto.DeriveKey(oldPassword, salt)
	newKey = crypto.DeriveKey(newPassword, salt)

	logger.Info("密码修改成功")
	return oldKey, newKey, nil
}

// GenerateSalt 生成随机盐值
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("生成盐值失败: %w", err)
	}
	return salt, nil
}

// EncodeSalt 编码盐值为字符串
func EncodeSalt(salt []byte) string {
	return base64.StdEncoding.EncodeToString(salt)
}

// DecodeSalt 解码盐值字符串
func DecodeSalt(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
