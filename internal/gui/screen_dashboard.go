package gui

import (
	"paradise/internal/logger"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showDashboard() {
	title := widget.NewLabelWithStyle("Paradise", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	vaultInfo := widget.NewLabelWithStyle("保险库: "+a.vaultDir, fyne.TextAlignCenter, fyne.TextStyle{Italic: true})

	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.Wrapping = fyne.TextWrapWord
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	encryptBtn := widget.NewButtonWithIcon("加密目录", theme.ConfirmIcon(), func() {
		showEncryptDialog(a.window, a.key, a.vaultDir, statusLabel)
	})
	encryptBtn.Importance = widget.MediumImportance

	decryptBtn := widget.NewButtonWithIcon("解密目录", theme.VisibilityIcon(), func() {
		showDecryptDialog(a.window, a.key, a.salt, a.vaultDir, statusLabel)
	})
	decryptBtn.Importance = widget.MediumImportance

	writeBackBtn := widget.NewButtonWithIcon("写回并清理", theme.UploadIcon(), func() {
		showWriteBackDialog(a.window, a.key, statusLabel)
	})
	writeBackBtn.Importance = widget.MediumImportance

	backupBtn := widget.NewButtonWithIcon("备份加密目录", theme.StorageIcon(), func() {
		showBackupDialog(a.window, a.vaultDir, statusLabel)
	})
	backupBtn.Importance = widget.MediumImportance

	restoreBtn := widget.NewButtonWithIcon("恢复备份", theme.DownloadIcon(), func() {
		showRestoreDialog(a.window, statusLabel)
	})
	restoreBtn.Importance = widget.MediumImportance

	changePwdBtn := widget.NewButtonWithIcon("修改密码", theme.SettingsIcon(), func() {
		a.showChangePasswordDialog(statusLabel)
	})
	changePwdBtn.Importance = widget.WarningImportance

	lockBtn := widget.NewButtonWithIcon("锁定", theme.CancelIcon(), func() {
		logger.Info("用户锁定应用")
		a.tryAutoWriteback(func() {
			a.password = ""
			a.key = nil
			a.salt = nil
			a.showLogin()
		})
	})
	lockBtn.Importance = widget.DangerImportance

	btnGrid := container.NewGridWithColumns(2,
		encryptBtn, decryptBtn,
		writeBackBtn, backupBtn,
		restoreBtn, changePwdBtn,
		lockBtn,
	)

	operationCard := widget.NewCard("操作", "", btnGrid)

	statusScroll := container.NewScroll(statusLabel)
	statusScroll.SetMinSize(fyne.NewSize(0, 80))

	bottomSection := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle("📋 操作日志", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		statusScroll,
	)

	topSection := container.NewVBox(
		title,
		vaultInfo,
		widget.NewSeparator(),
	)

	content := container.NewBorder(
		topSection,
		bottomSection,
		nil,
		nil,
		operationCard,
	)

	a.window.SetContent(container.NewPadded(content))
}
