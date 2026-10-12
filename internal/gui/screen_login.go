package gui

import (
	"paradise/internal/config"
	"paradise/internal/crypto"
	"paradise/internal/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showLogin() {
	title := widget.NewLabelWithStyle("Paradise", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabelWithStyle("文件保险库", fyne.TextAlignCenter, fyne.TextStyle{})

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("主密码")

	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter

	loginBtn := widget.NewButton("登 录", func() {
		pwd := passwordEntry.Text
		if err := config.VerifyPassword(pwd); err != nil {
			logger.Warn("登录失败：密码错误")
			statusLabel.SetText("密码错误，请重试")
			passwordEntry.SetText("")
			return
		}

		salt, err := config.LoadSalt()
		if err != nil {
			logger.Error("登录失败：加载盐值失败", "error", err)
			statusLabel.SetText("加载配置失败: " + err.Error())
			return
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			logger.Error("登录失败：加载配置失败", "error", err)
			statusLabel.SetText("加载配置失败: " + err.Error())
			return
		}

		a.password = pwd
		a.salt = salt
		a.key = crypto.DeriveKey(pwd, salt)
		a.vaultDir = cfg.VaultDir
		if a.vaultDir == "" {
			a.vaultDir = config.DefaultVaultDir
		}

		resolvedVault, err := config.ResolveVaultDir(a.vaultDir)
		if err != nil {
			logger.Error("登录失败：解析保险库路径失败", "error", err)
			statusLabel.SetText("解析保险库路径失败: " + err.Error())
			return
		}
		a.vaultDir = resolvedVault

		logger.Info("登录成功", "vault_dir", a.vaultDir)
		a.startupCleanup()
	})
	loginBtn.Importance = widget.HighImportance

	passwordEntry.OnSubmitted = func(text string) {
		loginBtn.OnTapped()
	}

	form := container.NewVBox(
		widget.NewLabelWithStyle("🔐 主密码", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		passwordEntry,
		statusLabel,
		loginBtn,
	)

	card := widget.NewCard("", "", form)

	content := container.NewBorder(
		nil,
		nil,
		nil,
		nil,
		container.NewVBox(
			title,
			subtitle,
			widget.NewSeparator(),
			card,
		),
	)

	a.window.SetContent(container.NewPadded(content))
}
