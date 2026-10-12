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

func showRestoreDialog(w fyne.Window, statusLabel *widget.Label) {
	// 备份目录选择
	backupDirLabel := widget.NewLabel("(未选择)")
	backupDirLabel.Truncation = fyne.TextTruncateClip

	backupDirBtn := widget.NewButton("浏览", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			backupDirLabel.SetText(uri.Path())
			backups, err := vault.ListBackups(uri.Path())
			if err != nil || len(backups) == 0 {
				backupDirLabel.SetText(uri.Path() + " (无备份文件)")
			}
		}, w)
	})

	// 备份文件列表
	backupList := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {},
	)

	var backupFiles []string
	selectedBackup := ""

	backupList.OnSelected = func(id widget.ListItemID) {
		if id < len(backupFiles) {
			selectedBackup = backupFiles[id]
		}
	}

	// 刷新备份列表
	refreshBackups := func(dir string) {
		var err error
		backupFiles, err = vault.ListBackups(dir)
		if err != nil {
			backupFiles = nil
		}
		backupList.Length = func() int { return len(backupFiles) }
		backupList.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }
		backupList.UpdateItem = func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(backupFiles) {
				obj.(*widget.Label).SetText(backupFiles[id])
			}
		}
		backupList.Refresh()
	}

	// 浏览备份目录时刷新列表
	backupDirBtn.OnTapped = func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			backupDirLabel.SetText(uri.Path())
			refreshBackups(uri.Path())
		}, w)
	}

	// 目标目录
	targetLabel, targetBtn := folderPickerRow(w, executableDir())

	var d dialog.Dialog

	confirmBtn := widget.NewButton("开始恢复", func() {
		backupDir := backupDirLabel.Text
		target := targetLabel.Text

		if backupDir == "(未选择)" || target == "(未选择)" {
			dialog.ShowError(errors.New("请选择备份目录和目标目录"), w)
			return
		}
		if selectedBackup == "" {
			dialog.ShowError(errors.New("请选择要恢复的备份文件"), w)
			return
		}

		backupFile := backupDir + "/" + selectedBackup
		dialog.ShowConfirm("确认恢复",
			fmt.Sprintf("将恢复备份到:\n%s\n\n备份文件: %s", target, selectedBackup),
			func(ok bool) {
				if !ok {
					return
				}
				logger.Info("GUI: 开始恢复备份", "backup_file", backupFile, "target", target)
				if err := vault.RestoreBackup(backupFile, target); err != nil {
					logger.Error("GUI: 恢复失败", "backup_file", backupFile, "error", err)
					dialog.ShowError(fmt.Errorf("恢复失败: %w", err), w)
					return
				}
				logger.Info("GUI: 恢复完成", "backup_file", backupFile, "target", target)
				statusLabel.SetText("✓ 恢复完成: " + target)
				d.Hide()
			}, w)
	})
	confirmBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("备份目录:", container.NewBorder(nil, nil, nil, backupDirBtn, backupDirLabel)),
		widget.NewFormItem("备份文件:", backupList),
		widget.NewFormItem("目标目录:", container.NewBorder(nil, nil, nil, targetBtn, targetLabel)),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("恢复备份", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		confirmBtn,
	)

	d = dialog.NewCustom("恢复备份", "取消", content, w)
	d.Resize(fyne.NewSize(450, 380))
	d.Show()
}
