package gui

import (
	"errors"
	"fmt"

	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func showBackupDialog(w fyne.Window, vaultDir string, statusLabel *widget.Label) {
	dstLabel, dstBtn := folderPickerRow(w, executableDir())

	vaultInfo := widget.NewLabelWithStyle("备份源: "+vaultDir, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	var d dialog.Dialog

	confirmBtn := widget.NewButton("开始备份", func() {
		dst := dstLabel.Text
		if dst == "(未选择)" {
			dialog.ShowError(errors.New("请选择备份存放目录"), w)
			return
		}
		logger.Info("GUI: 开始备份", "vault", vaultDir, "backup_dir", dst)
		backupFile, err := vault.CreateBackup(vaultDir, dst)
		if err != nil {
			logger.Error("GUI: 备份失败", "vault", vaultDir, "error", err)
			dialog.ShowError(fmt.Errorf("备份失败: %w", err), w)
			return
		}
		logger.Info("GUI: 备份完成", "backup_file", backupFile)
		statusLabel.SetText("✓ 备份完成: " + backupFile)
		d.Hide()
	})
	confirmBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("备份目录:", container.NewBorder(nil, nil, nil, dstBtn, dstLabel)),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("备份保险库", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		vaultInfo,
		confirmBtn,
	)

	d = dialog.NewCustom("备份", "取消", content, w)
	d.Resize(fyne.NewSize(450, 200))
	d.Show()
}
