package config

import (
	"encoding/json"
	"fmt"
	"os"
	"paradise/internal/logger"
	"path/filepath"
)

const (
	dataDirName     = ".paradise"
	configFileName  = "paradise_config.json"
	DefaultVaultDir = "vault"
)

// AppConfig 应用配置
type AppConfig struct {
	VaultDir     string `json:"vault_dir"`
	Salt         string `json:"salt"`
	PasswordHash string `json:"password_hash"`
}

// DataDir 返回可执行文件所在目录下的 .paradise 数据目录
func DataDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}
	execDir := filepath.Dir(execPath)
	dir := filepath.Join(execDir, dataDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("无法创建数据目录: %w", err)
	}
	return dir, nil
}

// ConfigFilePath 返回配置文件路径
func ConfigFilePath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// LoadConfig 加载配置
func LoadConfig() (*AppConfig, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("配置文件不存在，返回默认配置", "path", path)
			return &AppConfig{}, nil
		}
		logger.Error("读取配置文件失败", "path", path, "error", err)
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		logger.Warn("配置文件解析失败，返回默认配置", "path", path, "error", err)
		return &AppConfig{}, nil
	}
	logger.Debug("配置加载成功", "vault_dir", cfg.VaultDir)
	return &cfg, nil
}

// SaveConfig 保存配置
func SaveConfig(cfg *AppConfig) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		logger.Error("序列化配置失败", "error", err)
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		logger.Error("写入配置文件失败", "path", path, "error", err)
		return err
	}
	logger.Debug("配置保存成功", "vault_dir", cfg.VaultDir)
	return nil
}

// ResolveVaultDir 将保险库相对路径解析为绝对路径
func ResolveVaultDir(vaultDir string) (string, error) {
	if filepath.IsAbs(vaultDir) {
		return vaultDir, nil
	}
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("无法获取可执行文件路径: %w", err)
	}
	return filepath.Join(filepath.Dir(execPath), vaultDir), nil
}
