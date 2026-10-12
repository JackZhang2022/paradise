package vault

import (
	"fmt"
	"os"
	"paradise/internal/crypto"
	"paradise/internal/logger"
	"path/filepath"
	"time"
)

// DecryptDirectory 解密整个目录到目标目录
func DecryptDirectory(sourceDir, targetDir string, key []byte) error {
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

	logger.Info("开始解密目录", "source", absSource, "target", absTarget)

	if err := os.MkdirAll(absTarget, 0700); err != nil {
		logger.Error("创建目标目录失败", "target", absTarget, "error", err)
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	if err := decryptDir(absSource, absTarget, key); err != nil {
		logger.Error("解密目录失败", "source", absSource, "error", err)
		return err
	}

	logger.Info("解密目录完成", "source", absSource, "target", absTarget)
	return nil
}

func decryptDir(source, target string, key []byte) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("读取目录 %s 失败: %w", source, err)
	}

	for _, entry := range entries {
		// 跳过元数据文件
		if entry.Name() == dirMetaFile {
			continue
		}

		srcPath := filepath.Join(source, entry.Name())

		if entry.IsDir() {
			origName, err := readDirMeta(srcPath, key)
			if err != nil {
				return fmt.Errorf("读取目录元数据失败 %s: %w", srcPath, err)
			}
			dstPath := filepath.Join(target, origName)
			if err := os.MkdirAll(dstPath, 0700); err != nil {
				return fmt.Errorf("创建目录 %s 失败: %w", dstPath, err)
			}
			if err := decryptDir(srcPath, dstPath, key); err != nil {
				return err
			}
		} else {
			origName, plaintext, err := decryptFileWithName(srcPath, key)
			if err != nil {
				return fmt.Errorf("解密文件 %s 失败: %w", srcPath, err)
			}
			dstPath := filepath.Join(target, origName)
			if err := os.WriteFile(dstPath, plaintext, 0600); err != nil {
				return err
			}
		}
	}
	return nil
}

func decryptFileWithName(srcPath string, key []byte) (originalName string, plaintext []byte, err error) {
	ciphertext, err := os.ReadFile(srcPath)
	if err != nil {
		return "", nil, err
	}
	return crypto.DecryptFileWithName(ciphertext, key)
}

func readDirMeta(dirPath string, key []byte) (string, error) {
	metaPath := filepath.Join(dirPath, dirMetaFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", fmt.Errorf("读取元数据文件失败: %w", err)
	}
	return crypto.DecryptDirName(data, key)
}

// DecryptAndTrack 解密并记录状态
func DecryptAndTrack(sourceDir, targetDir, salt string, key []byte) error {
	if err := DecryptDirectory(sourceDir, targetDir, key); err != nil {
		return err
	}

	absSource, _ := filepath.Abs(sourceDir)
	absTarget, _ := filepath.Abs(targetDir)

	record := DecryptRecord{
		SourceDir:     absSource,
		DecryptedDir:  absTarget,
		DecryptedTime: time.Now(),
		Salt:          salt,
	}
	if err := AddRecord(record, key); err != nil {
		logger.Error("添加解密记录失败", "source", absSource, "error", err)
		return err
	}
	logger.Info("解密并记录状态完成", "source", absSource, "target", absTarget)
	return nil
}
