// Package gui 提供 Paradise 图形用户界面。
//
// 屏幕文件：
//   - screen_setup.go:          首次设置界面
//   - screen_login.go:          登录界面
//   - screen_dashboard.go:      主功能面板
//   - screen_change_password.go: 修改密码界面
//
// 对话框文件：
//   - dialog_encrypt.go:   加密目录对话框
//   - dialog_decrypt.go:   解密目录对话框
//   - dialog_writeback.go: 写回并清理对话框
//   - dialog_backup.go:    备份对话框
//   - dialog_restore.go:   恢复备份对话框
//
// 共享工具：
//   - operations.go:  folderPickerRow, friendlyError, openDirectory
//   - theme.go:       中文字体主题
package gui

import (
	"errors"
	"fmt"
	"strings"

	"paradise/internal/config"
	"paradise/internal/crypto"
	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// App 封装 Fyne 应用
type App struct {
	fyneApp       fyne.App
	window        fyne.Window
	password      string
	key           []byte
	salt          []byte
	vaultDir      string
	closeGuardSet bool
}

// Run 启动 GUI 应用
func Run() {
	a := &App{
		fyneApp: app.NewWithID("com.paradise.vault"),
	}
	a.fyneApp.Settings().SetTheme(newTheme())
	a.window = a.fyneApp.NewWindow("Paradise")
	a.window.Resize(fyne.NewSize(560, 540))
	a.setupCloseGuard()

	// 测试模式：自动初始化并登录，跳过设置/登录界面
	if config.IsTestMode() {
		logger.Info("测试模式已启用")
		if config.IsFirstRun() {
			if err := config.AutoSetupTest(); err != nil {
				logger.Error("测试模式自动初始化失败", "error", err)
			}
		}
		if err := a.autoLogin(config.TestPassword); err != nil {
			logger.Error("测试模式自动登录失败", "error", err)
			a.showLogin()
		} else {
			a.showDashboard()
		}
	} else if config.IsFirstRun() {
		logger.Info("首次运行，显示设置界面")
		a.showSetup()
	} else {
		logger.Info("显示登录界面")
		a.showLogin()
	}

	a.window.ShowAndRun()
}

// autoLogin 用指定密码自动登录，设置 App 的密钥和保险库目录
func (a *App) autoLogin(password string) error {
	if err := config.VerifyPassword(password); err != nil {
		return err
	}

	salt, err := config.LoadSalt()
	if err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	a.password = password
	a.salt = salt
	a.key = crypto.DeriveKey(password, salt)
	a.vaultDir = cfg.VaultDir
	if a.vaultDir == "" {
		a.vaultDir = config.DefaultVaultDir
	}

	resolvedVault, err := config.ResolveVaultDir(a.vaultDir)
	if err != nil {
		return err
	}
	a.vaultDir = resolvedVault

	logger.Info("自动登录成功", "vault_dir", a.vaultDir)
	return nil
}

// setupCloseGuard 设置窗口关闭拦截：关闭前自动写回所有已解密目录
func (a *App) setupCloseGuard() {
	if a.closeGuardSet {
		return
	}
	a.closeGuardSet = true
	a.window.SetCloseIntercept(func() {
		if a.key == nil {
			a.window.Close()
			return
		}
		a.tryAutoWriteback(a.window.Close)
	})
}

// tryAutoWriteback 尝试自动写回所有已解密目录，完成后调用 onDone
func (a *App) tryAutoWriteback(onDone func()) {
	records, err := vault.GetAllRecords(a.key)
	if err != nil || len(records) == 0 {
		onDone()
		return
	}

	logger.Info("自动写回解密目录", "records", len(records))

	// 先检查是否有文件被其他进程占用
	var busyFiles []string
	for _, rec := range records {
		files := vault.CheckFilesInUse(rec.DecryptedDir)
		for _, f := range files {
			busyFiles = append(busyFiles, f)
		}
	}
	if len(busyFiles) > 0 {
		a.showBusyFilesRetryDialog(busyFiles, onDone)
		return
	}

	a.doWriteback(records, onDone)
}

// doWriteback 执行实际写回操作
func (a *App) doWriteback(records []vault.DecryptRecord, onDone func()) {
	var failedDirs []string
	var failedErrors []string

	for _, rec := range records {
		logger.Info("自动写回", "source", rec.SourceDir, "decrypted", rec.DecryptedDir)
		err := vault.ReEncryptAndClean(rec.SourceDir, rec.DecryptedDir, a.key)
		if err != nil {
			if errors.Is(err, vault.ErrNothingToWriteBack) || errors.Is(err, vault.ErrDecryptedDirNotExist) {
				_ = vault.RemoveRecord(rec.SourceDir, a.key)
				logger.Info("解密目录已不存在，移除记录", "decrypted", rec.DecryptedDir)
				continue
			}
			logger.Warn("自动写回失败", "decrypted", rec.DecryptedDir, "error", err)
			failedDirs = append(failedDirs, rec.DecryptedDir)
			failedErrors = append(failedErrors, err.Error())
		}
	}

	if len(failedDirs) == 0 {
		logger.Info("所有解密目录已自动写回")
		onDone()
		return
	}

	a.showWritebackRetryDialog(failedDirs, failedErrors, onDone)
}

// showBusyFilesRetryDialog 文件被占用时显示重试对话框
func (a *App) showBusyFilesRetryDialog(files []string, onDone func()) {
	var msgLines []string
	msgLines = append(msgLines, "以下文件正在被其他程序占用，无法执行写回：")
	msgLines = append(msgLines, "")
	for _, f := range files {
		msgLines = append(msgLines, fmt.Sprintf("• %s", f))
	}
	msgLines = append(msgLines, "")
	msgLines = append(msgLines, "请关闭占用文件的程序后点击「重试」。")

	msgLabel := widget.NewLabel(strings.Join(msgLines, "\n"))
	msgLabel.Wrapping = fyne.TextWrapWord

	retryBtn := widget.NewButtonWithIcon("重试", theme.ViewRefreshIcon(), func() {
		logger.Info("用户选择重试写回")
		a.tryAutoWriteback(onDone)
	})
	retryBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("⚠ 文件被占用", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		msgLabel,
		retryBtn,
	)

	d := dialog.NewCustom("文件占用", "跳过", content, a.window)
	d.SetOnClosed(onDone)
	d.Resize(fyne.NewSize(500, 300))
	d.Show()
}

// showWritebackRetryDialog 显示写回失败的重试对话框，onDone 是清理完成后的回调
func (a *App) showWritebackRetryDialog(failedDirs []string, failedErrors []string, onDone func()) {
	var msgLines []string
	msgLines = append(msgLines, "以下解密目录写回后无法自动删除：")
	msgLines = append(msgLines, "")
	for i, d := range failedDirs {
		msgLines = append(msgLines, fmt.Sprintf("• %s", d))
		if i < len(failedErrors) && failedErrors[i] != "" {
			msgLines = append(msgLines, fmt.Sprintf("  原因: %s", failedErrors[i]))
		}
	}
	msgLines = append(msgLines, "")
	msgLines = append(msgLines, "请关闭占用这些目录的程序后点击「重试」。")

	msgLabel := widget.NewLabel(strings.Join(msgLines, "\n"))
	msgLabel.Wrapping = fyne.TextWrapWord

	retryBtn := widget.NewButtonWithIcon("重试", theme.ViewRefreshIcon(), func() {
		logger.Info("用户选择重试自动写回")
		a.tryAutoWriteback(onDone)
	})
	retryBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("⚠ 自动写回部分失败", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		msgLabel,
		retryBtn,
	)

	d := dialog.NewCustom("自动清理", "跳过", content, a.window)
	d.SetOnClosed(onDone)
	d.Resize(fyne.NewSize(500, 300))
	d.Show()
}
