package gui

import (
	"errors"
	"fmt"

	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func showWriteBackDialog(w fyne.Window, key []byte, statusLabel *widget.Label) {
	records, err := vault.GetAllRecords(key)
	if err != nil || len(records) == 0 {
		dialog.ShowInformation("提示", "没有已解密的目录记录", w)
		return
	}

	listWidget := widget.NewList(
		func() int { return len(records) },
		func() fyne.CanvasObject {
			srcLabel := widget.NewLabel("")
			srcLabel.Truncation = fyne.TextTruncateClip
			dstLabel := widget.NewLabel("")
			dstLabel.Truncation = fyne.TextTruncateClip
			dstLabel.TextStyle = fyne.TextStyle{Italic: true}
			return container.NewVBox(srcLabel, dstLabel)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			srcLabel := c.Objects[0].(*widget.Label)
			dstLabel := c.Objects[1].(*widget.Label)
			srcLabel.SetText("加密目录: " + records[id].SourceDir)
			dstLabel.SetText("解密目录: " + records[id].DecryptedDir)
		},
	)

	selectedIdx := -1
	listWidget.OnSelected = func(id widget.ListItemID) {
		selectedIdx = id
	}

	var d dialog.Dialog

	confirmBtn := widget.NewButtonWithIcon("写回并清理", theme.UploadIcon(), func() {
		if selectedIdx < 0 || selectedIdx >= len(records) {
			dialog.ShowError(errors.New("请先选择一条记录"), w)
			return
		}
		rec := records[selectedIdx]
		dialog.ShowConfirm("确认操作",
			fmt.Sprintf("将把解密目录重新加密写回原目录，并删除解密目录:\n\n%s", rec.DecryptedDir),
			func(ok bool) {
				if !ok {
					return
				}
				logger.Info("GUI: 开始写回并清理", "source", rec.SourceDir, "decrypted", rec.DecryptedDir)
				if err := vault.ReEncryptAndClean(rec.SourceDir, rec.DecryptedDir, key); err != nil {
					logger.Error("GUI: 写回失败", "error", err)
					dialog.ShowError(errors.New(friendlyError(err, "写回")), w)
					return
				}
				logger.Info("GUI: 写回并清理完成", "source", rec.SourceDir)
				statusLabel.SetText("✓ 写回完成，解密目录已删除")

				// 刷新列表
				records, err = vault.GetAllRecords(key)
				if err != nil || len(records) == 0 {
					d.Hide()
					return
				}
				selectedIdx = -1
				listWidget.Refresh()
			}, w)
	})
	confirmBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		widget.NewLabelWithStyle("写回并清理", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("选择要写回的已解密目录:"),
		listWidget,
		confirmBtn,
	)

	d = dialog.NewCustom("写回并清理", "取消", content, w)
	d.Resize(fyne.NewSize(550, 350))
	d.Show()
}
