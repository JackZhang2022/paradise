package vault

import (
	"fmt"
	"os"
	"paradise/internal/logger"
	"path/filepath"
)

// ReEncryptAndClean 将解密后的目录重新加密写回保险库，并删除解密目录
// sourceDir: 保险库根目录
// decryptedDir: 已解密的目录
func ReEncryptAndClean(sourceDir, decryptedDir string, key []byte) error {
	absVault, err := filepath.Abs(sourceDir)
	if err != nil {
		logger.Error("解析源路径失败", "source", sourceDir, "error", err)
		return fmt.Errorf("解析源路径失败: %w", err)
	}
	absDecrypted, err := filepath.Abs(decryptedDir)
	if err != nil {
		logger.Error("解析解密路径失败", "decrypted", decryptedDir, "error", err)
		return fmt.Errorf("解析解密路径失败: %w", err)
	}

	logger.Info("开始写回并清理", "vault", absVault, "decrypted", absDecrypted)

	if _, err := os.Stat(absDecrypted); os.IsNotExist(err) {
		logger.Warn("解密目录不存在", "dir", absDecrypted)
		return fmt.Errorf("%w: %s", ErrDecryptedDirNotExist, absDecrypted)
	}

	// 遍历保险库中的加密子目录，逐个写回
	vaultEntries, err := os.ReadDir(absVault)
	if err != nil {
		logger.Error("读取保险库目录失败", "dir", absVault, "error", err)
		return fmt.Errorf("读取保险库目录失败: %w", err)
	}

	processed := 0
	for _, entry := range vaultEntries {
		if !entry.IsDir() {
			continue
		}

		encSubDir := filepath.Join(absVault, entry.Name())

		// 读取原始目录名
		origName, err := readDirMeta(encSubDir, key)
		if err != nil {
			logger.Warn("读取目录元数据失败，跳过", "dir", encSubDir, "error", err)
			continue
		}

		decSubDir := filepath.Join(absDecrypted, origName)
		if _, err := os.Stat(decSubDir); os.IsNotExist(err) {
			logger.Warn("解密子目录不存在，跳过", "dir", decSubDir)
			continue
		}

		// 在删除任何数据前，先检查解密子目录中是否有文件被其他进程占用
		if busy := CheckFilesInUse(decSubDir); len(busy) > 0 {
			logger.Warn("解密子目录中有文件被占用，跳过写回", "subdir", origName, "busy_count", len(busy))
			continue
		}

		// 删除旧的加密子目录
		if err := os.RemoveAll(encSubDir); err != nil {
			logger.Error("删除旧加密子目录失败", "dir", encSubDir, "error", err)
			return fmt.Errorf("删除旧加密子目录失败: %w", err)
		}

		// 重新创建加密子目录并写入内容
		if err := os.MkdirAll(encSubDir, 0700); err != nil {
			logger.Error("创建加密子目录失败", "dir", encSubDir, "error", err)
			return fmt.Errorf("创建加密子目录失败: %w", err)
		}
		if err := encryptDir(decSubDir, encSubDir, key); err != nil {
			logger.Error("重新加密失败", "subdir", origName, "error", err)
			return fmt.Errorf("重新加密失败: %w", err)
		}
		if err := createDirMeta(encSubDir, origName, key); err != nil {
			logger.Error("写入元数据失败", "subdir", origName, "error", err)
			return fmt.Errorf("写入元数据失败: %w", err)
		}

		// 删除解密子目录
		if err := os.RemoveAll(decSubDir); err != nil {
			logger.Error("删除解密子目录失败", "dir", decSubDir, "error", err)
			return fmt.Errorf("删除解密子目录失败: %w", err)
		}

		logger.Info("写回子目录完成", "subdir", origName, "vault_subdir", encSubDir)
		processed++
	}

	if processed == 0 {
		logger.Warn("没有找到可写回的解密子目录")
		return ErrNothingToWriteBack
	}

	// 移除状态记录
	if err := RemoveRecord(absVault, key); err != nil {
		logger.Error("移除状态记录失败", "source", absVault, "error", err)
		return fmt.Errorf("移除状态记录失败: %w", err)
	}

	logger.Info("写回并清理完成", "vault", absVault, "decrypted", absDecrypted, "processed", processed)
	return nil
}
