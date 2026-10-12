package vault

import (
	"fmt"
	"os"
	"paradise/internal/crypto"
	"paradise/internal/logger"
	"path/filepath"
)

// ReEncryptVault 用新密钥重新加密保险库中的所有文件
func ReEncryptVault(vaultDir string, oldKey, newKey []byte) error {
	absVault, err := filepath.Abs(vaultDir)
	if err != nil {
		logger.Error("解析保险库路径失败", "vault", vaultDir, "error", err)
		return fmt.Errorf("解析保险库路径失败: %w", err)
	}

	logger.Info("开始重新加密保险库", "vault", absVault)

	if _, err := os.Stat(absVault); os.IsNotExist(err) {
		logger.Info("保险库不存在，跳过重加密", "vault", absVault)
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "paradise-rekey-*")
	if err != nil {
		logger.Error("创建临时目录失败", "error", err)
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := DecryptDirectory(absVault, tmpDir, oldKey); err != nil {
		logger.Error("解密保险库失败", "error", err)
		return fmt.Errorf("解密保险库失败: %w", err)
	}

	// 清空保险库目录
	entries, err := os.ReadDir(absVault)
	if err != nil {
		logger.Error("读取保险库目录失败", "error", err)
		return fmt.Errorf("读取保险库目录失败: %w", err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(absVault, entry.Name())); err != nil {
			logger.Error("删除旧文件失败", "file", entry.Name(), "error", err)
			return fmt.Errorf("删除旧文件失败: %w", err)
		}
	}

	// 对临时目录中每个顶层目录，用新密钥重新加密回保险库
	tmpEntries, err := os.ReadDir(tmpDir)
	if err != nil {
		logger.Error("读取临时目录失败", "error", err)
		return fmt.Errorf("读取临时目录失败: %w", err)
	}
	for _, entry := range tmpEntries {
		if !entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(tmpDir, entry.Name())
		encName := crypto.DeriveEncryptedName(entry.Name(), newKey)
		dstPath := filepath.Join(absVault, encName)
		if err := os.MkdirAll(dstPath, 0700); err != nil {
			logger.Error("创建加密子目录失败", "dir", dstPath, "error", err)
			return fmt.Errorf("创建加密子目录失败: %w", err)
		}
		if err := encryptDir(srcPath, dstPath, newKey); err != nil {
			logger.Error("重新加密目录失败", "dir", entry.Name(), "error", err)
			return fmt.Errorf("重新加密目录失败: %w", err)
		}
		if err := createDirMeta(dstPath, entry.Name(), newKey); err != nil {
			logger.Error("写入元数据失败", "dir", dstPath, "error", err)
			return fmt.Errorf("写入元数据失败: %w", err)
		}
	}

	logger.Info("重新加密保险库完成", "vault", absVault)
	return nil
}
