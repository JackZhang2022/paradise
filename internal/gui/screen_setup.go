package gui

import (
	"paradise/internal/config"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showSetup() {
	title := widget.NewLabelWithStyle("Paradise", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabelWithStyle("首次设置 - 请设置主密码和保险库目录", fyne.TextAlignCenter, fyne.TextStyle{})

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("输入密码（至少 6 位）")

	confirmEntry := widget.NewPasswordEntry()
	confirmEntry.SetPlaceHolder("确认密码")

	vaultEntry := widget.NewEntry()
	vaultEntry.SetPlaceHolder("保险库目录名称")
	vaultEntry.SetText(config.DefaultVaultDir)

	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter

	submitBtn := widget.NewButton("初始化", func() {
		pwd := passwordEntry.Text
		confirm := confirmEntry.Text
		vaultName := vaultEntry.Text

		if pwd != confirm {
			statusLabel.SetText("两次输入的密码不一致")
			return
		}
		if vaultName == "" {
			vaultName = config.DefaultVaultDir
		}

		if err := config.SetPassword(pwd); err != nil {
			statusLabel.SetText(err.Error())
			return
		}

		cfg, _ := config.LoadConfig()
		if cfg == nil {
			cfg = &config.AppConfig{}
		}
		cfg.VaultDir = vaultName
		if err := config.SaveConfig(cfg); err != nil {
			statusLabel.SetText("保存保险库配置失败: " + err.Error())
			return
		}

		statusLabel.SetText("初始化成功！请登录")
		a.showLogin()
	})
	submitBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewLabelWithStyle("🔐 主密码", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		passwordEntry,
		confirmEntry,
		widget.NewLabelWithStyle("📁 保险库", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		vaultEntry,
		widget.NewLabelWithStyle("保险库将保存在程序同目录下", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
		statusLabel,
		submitBtn,
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
