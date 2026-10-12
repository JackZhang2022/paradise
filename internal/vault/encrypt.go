package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"paradise/internal/crypto"
	"paradise/internal/logger"
)

const dirMetaFile = ".paradise_name"

// EncryptDirectory 加密整个目录到目标目录，保留源目录名
// 例如: EncryptDirectory("/home/user/myproject", "vault/", key)
// 结果: vault/<HMAC_of_myproject>/
//
//	├── .paradise_name  (加密的 "myproject")
//	├── <HMAC of main.go>
//	└── <HMAC of src>/
func EncryptDirectory(sourceDir, targetDir string, key []byte) error {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		logger.Error("解析源路径失败", "source", sourceDir, "error", err)
		return fmt.Errorf("解析源路径失败: %w", err)
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		logger.Error("解析目标路径失败", "target", targetDir, "error", err)
		return fmt.Errorf("解析目标路径失败: %w", err)
	}

	logger.Info("开始加密目录", "source", absSource, "target", absTarget)

	// 先检查源目录是否存在，避免删除旧的加密子目录后才发现源目录不存在
	srcInfo, err := os.Stat(absSource)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error("源目录不存在", "source", absSource)
			return fmt.Errorf("%w: %s", ErrSourceNotExist, absSource)
		}
		logger.Error("读取源目录失败", "source", absSource, "error", err)
		return fmt.Errorf("读取源目录失败: %w", err)
	}
	if !srcInfo.IsDir() {
		logger.Error("源路径不是目录", "source", absSource)
		return fmt.Errorf("%w: %s", ErrSourceNotDir, absSource)
	}

	if err := os.MkdirAll(absTarget, 0700); err != nil {
		logger.Error("创建目标目录失败", "target", absTarget, "error", err)
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 用源目录名在目标下创建加密子目录，保留顶层目录名
	srcName := filepath.Base(absSource)
	encName := crypto.DeriveEncryptedName(srcName, key)
	vaultSubDir := filepath.Join(absTarget, encName)

	// 如果同名加密子目录已存在，先删除旧内容避免残留
	if _, err := os.Stat(vaultSubDir); err == nil {
		logger.Info("同名加密子目录已存在，先删除", "dir", vaultSubDir)
		if err := os.RemoveAll(vaultSubDir); err != nil {
			logger.Error("删除旧加密子目录失败", "dir", vaultSubDir, "error", err)
			return fmt.Errorf("删除旧加密子目录失败: %w", err)
		}
	}

	if err := os.MkdirAll(vaultSubDir, 0700); err != nil {
		logger.Error("创建保险库子目录失败", "dir", vaultSubDir, "error", err)
		return fmt.Errorf("创建保险库子目录失败: %w", err)
	}

	if err := encryptDir(absSource, vaultSubDir, key); err != nil {
		logger.Error("加密目录失败", "source", absSource, "error", err)
		return err
	}
	if err := createDirMeta(vaultSubDir, srcName, key); err != nil {
		logger.Error("写入目录元数据失败", "dir", vaultSubDir, "error", err)
		return err
	}

	logger.Info("加密目录完成", "source", absSource, "vault_subdir", vaultSubDir)

	// 删除原始目录
	if err := os.RemoveAll(absSource); err != nil {
		logger.Warn("删除原始目录失败", "dir", absSource, "error", err)
	}
	return nil
}

func encryptDir(source, target string, key []byte) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("读取目录 %s 失败: %w", source, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(source, entry.Name())

		if entry.IsDir() {
			encName := crypto.DeriveEncryptedName(entry.Name(), key)
			dstPath := filepath.Join(target, encName)
			if err := os.MkdirAll(dstPath, 0700); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			if err := encryptDir(srcPath, dstPath, key); err != nil {
				return err
			}
			if err := createDirMeta(dstPath, entry.Name(), key); err != nil {
				return fmt.Errorf("写入目录元数据失败: %w", err)
			}
		} else {
			if err := encryptFileWithName(srcPath, target, entry.Name(), key); err != nil {
				return fmt.Errorf("加密文件 %s 失败: %w", srcPath, err)
			}
		}
	}
	return nil
}

func encryptFileWithName(srcPath, dstDir, originalName string, key []byte) error {
	plaintext, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	encrypted, err := crypto.EncryptFileWithName(plaintext, originalName, key)
	if err != nil {
		return err
	}
	encName := crypto.DeriveEncryptedName(originalName, key)
	return os.WriteFile(filepath.Join(dstDir, encName), encrypted, 0600)
}

func createDirMeta(dirPath, originalName string, key []byte) error {
	_, metadata, err := crypto.EncryptDirName(originalName, key)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dirPath, dirMetaFile), metadata, 0600)
}
