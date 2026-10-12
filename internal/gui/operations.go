// Package gui - 对话框共享工具
//
// 本文件包含所有对话框共用的辅助函数：
//   - folderPickerRow: 目录选择行组件
//   - friendlyError: 技术错误转用户友好提示
//
// 各对话框实现已拆分到独立文件：
//   - dialog_encrypt.go
//   - dialog_decrypt.go
//   - dialog_writeback.go
//   - dialog_backup.go
//   - dialog_restore.go
package gui

import (
	"errors"
	"os"
	"os/exec"
	"paradise/internal/logger"
	"paradise/internal/vault"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// folderPickerRow 创建一行：标签 + 路径显示 + 浏览按钮
// startDir 是文件选择器的初始目录
func folderPickerRow(w fyne.Window, startDir string) (*widget.Label, *widget.Button) {
	pathLabel := widget.NewLabel("(未选择)")
	pathLabel.Truncation = fyne.TextTruncateClip

	browseBtn := widget.NewButton("浏览", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			pathLabel.SetText(uri.Path())
		}, w)
		if startDir != "" {
			if u, err := storage.ListerForURI(storage.NewFileURI(startDir)); err == nil {
				d.SetLocation(u)
			}
		}
		d.Resize(fyne.NewSize(600, 400))
		d.Show()
	})

	return pathLabel, browseBtn
}

// executableDir 返回可执行文件所在目录
func executableDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(execPath)
}

// openDirectory 在系统文件管理器中打开指定目录
func openDirectory(path string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		args = []string{path}
	case "darwin":
		cmd = "open"
		args = []string{path}
	case "windows":
		cmd = "explorer"
		args = []string{path}
	default:
		logger.Warn("不支持的操作系统，无法自动打开目录", "os", runtime.GOOS)
		return
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		logger.Warn("自动打开目录失败", "path", path, "error", err)
	}
}

// friendlyError 将技术性错误转化为用户友好的提示
func friendlyError(err error, operation string) string {
	switch {
	case errors.Is(err, vault.ErrSourceNotExist):
		return "操作失败：选择的目录不存在，请确认目录路径正确"
	case errors.Is(err, vault.ErrSourceNotDir):
		return "操作失败：选择的路径不是一个目录"
	case errors.Is(err, vault.ErrDecryptedDirNotExist):
		return "操作失败：解密目标目录不存在，请先解密或重新选择"
	case errors.Is(err, vault.ErrNothingToWriteBack),
		errors.Is(err, vault.ErrReadDirMeta):
		return "操作失败：保险库中没有可解密的内容"
	case errors.Is(err, vault.ErrCreateDir):
		return "操作失败：无法创建目录，请检查磁盘空间和权限"
	case errors.Is(err, vault.ErrEncrypt):
		return "操作失败：加密过程中出现错误，请重试"
	case errors.Is(err, vault.ErrDecrypt):
		return "操作失败：解密过程中出现错误，请确认密码正确"
	case errors.Is(err, vault.ErrWriteBack):
		return "操作失败：写回过程中出现错误，请重试"
	default:
		return operation + "失败：" + err.Error()
	}
}
