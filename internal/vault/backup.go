package vault

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"paradise/internal/logger"
	"path/filepath"
	"strings"
	"time"
)

// CreateBackup 创建加密目录的 tar.gz 备份
func CreateBackup(sourceDir, backupDir string) (string, error) {
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		logger.Error("解析源路径失败", "source", sourceDir, "error", err)
		return "", fmt.Errorf("解析源路径失败: %w", err)
	}
	absBackup, err := filepath.Abs(backupDir)
	if err != nil {
		logger.Error("解析备份路径失败", "backup", backupDir, "error", err)
		return "", fmt.Errorf("解析备份路径失败: %w", err)
	}

	logger.Info("开始创建备份", "source", absSource, "backup_dir", absBackup)

	if err := os.MkdirAll(absBackup, 0700); err != nil {
		logger.Error("创建备份目录失败", "dir", absBackup, "error", err)
		return "", fmt.Errorf("创建备份目录失败: %w", err)
	}

	baseName := filepath.Base(absSource)
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(absBackup, fmt.Sprintf("%s_%s.tar.gz", baseName, timestamp))

	file, err := os.Create(backupFile)
	if err != nil {
		logger.Error("创建备份文件失败", "file", backupFile, "error", err)
		return "", fmt.Errorf("创建备份文件失败: %w", err)
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	if err := addToTar(tw, absSource, ""); err != nil {
		logger.Error("打包失败", "source", absSource, "error", err)
		return "", fmt.Errorf("打包失败: %w", err)
	}

	logger.Info("备份创建完成", "backup_file", backupFile)
	return backupFile, nil
}

func addToTar(tw *tar.Writer, sourceDir, prefix string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(sourceDir, entry.Name())
		tarPath := filepath.Join(prefix, entry.Name())

		info, err := entry.Info()
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = tarPath

		if entry.IsDir() {
			header.Name += "/"
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if err := addToTar(tw, srcPath, tarPath); err != nil {
				return err
			}
		} else {
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			file, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, file); err != nil {
				file.Close()
				return err
			}
			file.Close()
		}
	}
	return nil
}

// RestoreBackup 从备份文件恢复到指定目录
func RestoreBackup(backupFile, targetDir string) error {
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		logger.Error("解析目标路径失败", "target", targetDir, "error", err)
		return fmt.Errorf("解析目标路径失败: %w", err)
	}

	logger.Info("开始恢复备份", "backup_file", backupFile, "target", absTarget)

	if err := os.MkdirAll(absTarget, 0700); err != nil {
		logger.Error("创建目标目录失败", "dir", absTarget, "error", err)
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	file, err := os.Open(backupFile)
	if err != nil {
		logger.Error("打开备份文件失败", "file", backupFile, "error", err)
		return fmt.Errorf("打开备份文件失败: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		logger.Error("创建 gzip reader 失败", "error", err)
		return fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("读取 tar 条目失败", "error", err)
			return fmt.Errorf("读取 tar 条目失败: %w", err)
		}

		targetPath := filepath.Join(absTarget, header.Name)

		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(absTarget)) {
			logger.Warn("检测到非法路径，拒绝恢复", "path", header.Name)
			return fmt.Errorf("非法路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0700); err != nil {
				logger.Error("恢复时创建目录失败", "dir", targetPath, "error", err)
				return fmt.Errorf("创建目录失败: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0700); err != nil {
				logger.Error("恢复时创建父目录失败", "dir", filepath.Dir(targetPath), "error", err)
				return fmt.Errorf("创建父目录失败: %w", err)
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				logger.Error("恢复时创建文件失败", "file", targetPath, "error", err)
				return fmt.Errorf("创建文件失败: %w", err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				logger.Error("恢复时写入文件失败", "file", targetPath, "error", err)
				return fmt.Errorf("写入文件失败: %w", err)
			}
			outFile.Close()
		}
	}

	logger.Info("备份恢复完成", "backup_file", backupFile, "target", absTarget)
	return nil
}

// ListBackups 列出备份目录中的所有备份文件
func ListBackups(backupDir string) ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar.gz") {
			backups = append(backups, entry.Name())
		}
	}
	return backups, nil
}
