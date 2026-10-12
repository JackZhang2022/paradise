package gui

import (
	"errors"
	"fmt"

	"paradise/internal/config"
	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func (a *App) showChangePasswordDialog(statusLabel *widget.Label) {
	oldPwdEntry := widget.NewPasswordEntry()
	oldPwdEntry.SetPlaceHolder("输入旧密码")

	newPwdEntry := widget.NewPasswordEntry()
	newPwdEntry.SetPlaceHolder("输入新密码（至少 6 位）")

	confirmPwdEntry := widget.NewPasswordEntry()
	confirmPwdEntry.SetPlaceHolder("确认新密码")

	form := widget.NewForm(
		widget.NewFormItem("旧密码:", oldPwdEntry),
		widget.NewFormItem("新密码:", newPwdEntry),
		widget.NewFormItem("确认新密码:", confirmPwdEntry),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("修改密码", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("修改密码后，保险库中所有文件将用新密码重新加密。"),
		form,
	)

	d := dialog.NewCustomConfirm("修改密码", "确认修改", "取消", content, func(ok bool) {
		if !ok {
			return
		}
		oldPwd := oldPwdEntry.Text
		newPwd := newPwdEntry.Text
		confirmPwd := confirmPwdEntry.Text

		if newPwd != confirmPwd {
			logger.Warn("修改密码失败：两次输入的新密码不一致")
			dialog.ShowError(errors.New("两次输入的新密码不一致"), a.window)
			return
		}

		logger.Info("开始修改密码")
		// 执行密码修改
		oldKey, newKey, err := config.ChangePassword(oldPwd, newPwd)
		if err != nil {
			logger.Error("修改密码失败", "error", err)
			dialog.ShowError(err, a.window)
			return
		}

		// 重加密状态文件
		if err := vault.ReEncryptState(oldKey, newKey); err != nil {
			logger.Error("状态文件重加密失败", "error", err)
			dialog.ShowError(fmt.Errorf("状态文件重加密失败: %w", err), a.window)
			return
		}

		// 重加密保险库
		if err := vault.ReEncryptVault(a.vaultDir, oldKey, newKey); err != nil {
			logger.Error("保险库重加密失败", "error", err)
			dialog.ShowError(fmt.Errorf("保险库重加密失败: %w", err), a.window)
			return
		}

		// 更新当前密钥
		a.key = newKey
		a.password = newPwd

		logger.Info("密码修改成功")
		statusLabel.SetText("✓ 密码修改成功，所有文件已重新加密")
		dialog.ShowInformation("完成", "密码修改成功！保险库中所有文件已用新密码重新加密。", a.window)
	}, a.window)

	d.Resize(fyne.NewSize(420, 320))
	d.Show()
}
