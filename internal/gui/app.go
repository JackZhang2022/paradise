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
	a.showWritebackConfirmDialog(records, onDone)
}

// doWriteback 执行实际写回操作
func (a *App) doWriteback(records []vault.DecryptRecord, onDone func()) {
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
		}
	}

	logger.Info("所有解密目录已自动写回")
	onDone()
}

// showWritebackConfirmDialog 关闭前确认写回对话框
func (a *App) showWritebackConfirmDialog(records []vault.DecryptRecord, onDone func()) {
	msgLabel := widget.NewLabel("关闭前将自动写回并删除以下解密目录：\n\n请确保其中的文件已全部保存，否则修改将会丢失。")
	msgLabel.Wrapping = fyne.TextWrapWord

	dirList := widget.NewList(
		func() int { return len(records) },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(records[id].DecryptedDir)
		},
	)

	var d dialog.Dialog

	confirmBtn := widget.NewButtonWithIcon("确认写回", theme.ConfirmIcon(), func() {
		logger.Info("用户确认自动写回")
		d.Hide()
		a.doWriteback(records, onDone)
	})
	confirmBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButton("取消", func() {
		logger.Info("用户取消自动写回")
		d.Hide()
	})

	btns := container.NewHBox(
		cancelBtn,
		widget.NewLabel(""),
		confirmBtn,
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("确认写回并清理", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		msgLabel,
		dirList,
		btns,
	)

	d = dialog.NewCustom("关闭确认", "", content, a.window)
	d.Resize(fyne.NewSize(480, 320))
	d.Show()
}
