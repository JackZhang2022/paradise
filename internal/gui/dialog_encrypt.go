package gui

import (
	"errors"

	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func showEncryptDialog(w fyne.Window, key []byte, vaultDir string, statusLabel *widget.Label) {
	srcLabel, srcBtn := folderPickerRow(w)

	// 恢复上次加密的源目录
	if lastDir := vault.GetLastEncryptDir(key); lastDir != "" {
		srcLabel.SetText(lastDir)
	}

	// 目标固定为保险库目录
	vaultInfo := widget.NewLabelWithStyle("加密后存入保险库: "+vaultDir, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	var d dialog.Dialog

	confirmBtn := widget.NewButton("开始加密", func() {
		src := srcLabel.Text
		if src == "(未选择)" {
			dialog.ShowError(errors.New("请选择要加密的源目录"), w)
			return
		}
		logger.Info("GUI: 开始加密目录", "source", src, "vault", vaultDir)
		if err := vault.EncryptDirectory(src, vaultDir, key); err != nil {
			logger.Error("GUI: 加密失败", "source", src, "error", err)
			dialog.ShowError(errors.New(friendlyError(err, "加密")), w)
			return
		}
		logger.Info("GUI: 加密完成", "source", src)
		statusLabel.SetText("✓ 加密完成: " + src)

		// 记住本次加密的源目录
		_ = vault.SetLastEncryptDir(src, key)

		d.Hide()
	})
	confirmBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("源目录:", container.NewBorder(nil, nil, nil, srcBtn, srcLabel)),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("加密目录到保险库", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		vaultInfo,
		confirmBtn,
	)

	d = dialog.NewCustom("加密目录", "取消", content, w)
	d.Resize(fyne.NewSize(450, 200))
	d.Show()
}
