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
	"os"
	"path/filepath"
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
			a.ensureTestData()
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

// ensureTestData 测试模式下若保险库为空，预置一个加密测试目录
func (a *App) ensureTestData() {
	entries, err := os.ReadDir(a.vaultDir)
	if err != nil || len(entries) > 0 {
		return
	}

	logger.Info("测试模式：保险库为空，预置测试数据")

	testDir := filepath.Join(os.TempDir(), "paradise-test-data")
	os.RemoveAll(testDir)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0700)

	os.WriteFile(filepath.Join(testDir, "机密文件.txt"), []byte("这是测试机密文件内容。\n测试第二行。\n"), 0600)
	os.WriteFile(filepath.Join(testDir, "subdir", "notes.md"), []byte("# 测试笔记\n\n- 条目1\n- 条目2\n"), 0600)

	if err := vault.EncryptDirectory(testDir, a.vaultDir, a.key); err != nil {
		logger.Error("测试模式：预置测试数据失败", "error", err)
		return
	}
	logger.Info("测试模式：预置测试数据完成", "vault", a.vaultDir)
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
		a.tryAutoWriteback()
	})
}

// tryAutoWriteback 尝试自动写回所有已解密目录，失败则弹出重试对话框
func (a *App) tryAutoWriteback() {
	records, err := vault.GetAllRecords(a.key)
	if err != nil || len(records) == 0 {
		a.window.Close()
		return
	}

	logger.Info("关闭前自动写回", "records", len(records))

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
		a.window.Close()
		return
	}

	a.showWritebackRetryDialog(failedDirs, failedErrors)
}

// showWritebackRetryDialog 显示写回失败的重试对话框
func (a *App) showWritebackRetryDialog(failedDirs []string, failedErrors []string) {
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
	msgLines = append(msgLines, "请关闭占用这些目录的程序后点击「重试」，")
	msgLines = append(msgLines, "或选择「直接关闭」（目录将保留，需手动清理）。")

	msgLabel := widget.NewLabel(strings.Join(msgLines, "\n"))
	msgLabel.Wrapping = fyne.TextWrapWord

	retryBtn := widget.NewButtonWithIcon("重试", theme.ViewRefreshIcon(), func() {
		logger.Info("用户选择重试自动写回")
		a.tryAutoWriteback()
	})
	retryBtn.Importance = widget.HighImportance

	closeBtn := widget.NewButtonWithIcon("直接关闭", theme.CancelIcon(), func() {
		logger.Info("用户选择直接关闭，跳过自动写回")
		a.window.Close()
	})
	closeBtn.Importance = widget.DangerImportance

	btnBox := container.NewHBox(retryBtn, closeBtn)

	content := container.NewVBox(
		widget.NewLabelWithStyle("⚠ 自动写回部分失败", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		msgLabel,
		btnBox,
	)

	d := dialog.NewCustom("关闭确认", "取消", content, a.window)
	d.Resize(fyne.NewSize(500, 300))
	d.Show()
}
