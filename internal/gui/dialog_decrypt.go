package gui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"paradise/internal/config"
	"paradise/internal/logger"
	"paradise/internal/vault"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func showDecryptDialog(w fyne.Window, key []byte, salt []byte, vaultDir string, statusLabel *widget.Label) {
	dstLabel, dstBtn := folderPickerRow(w)

	// 恢复上次解密的目标目录
	if lastDir := vault.GetLastDecryptDir(key); lastDir != "" {
		dstLabel.SetText(lastDir)
	}

	// 源固定为保险库目录
	vaultInfo := widget.NewLabelWithStyle("从保险库解密: "+vaultDir, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	var d dialog.Dialog

	confirmBtn := widget.NewButton("开始解密", func() {
		dst := dstLabel.Text
		if dst == "(未选择)" {
			dialog.ShowError(errors.New("请选择解密目标目录"), w)
			return
		}
		saltStr := config.EncodeSalt(salt)
		logger.Info("GUI: 开始解密目录", "vault", vaultDir, "target", dst)
		if err := vault.DecryptAndTrack(vaultDir, dst, saltStr, key); err != nil {
			logger.Error("GUI: 解密失败", "vault", vaultDir, "error", err)
			dialog.ShowError(errors.New(friendlyError(err, "解密")), w)
			return
		}
		logger.Info("GUI: 解密完成", "vault", vaultDir, "target", dst)

		// 记住本次解密的目标目录
		_ = vault.SetLastDecryptDir(dst, key)

		statusLabel.SetText("✓ 解密完成: " + vaultDir + " → " + dst)
		// 打开解密后的子目录（若只有一个则直接进入，否则打开目标目录）
		openPath := dst
		if entries, err := os.ReadDir(dst); err == nil {
			var subdirs []string
			for _, e := range entries {
				if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					subdirs = append(subdirs, e.Name())
				}
			}
			if len(subdirs) == 1 {
				openPath = filepath.Join(dst, subdirs[0])
			}
		}
		openDirectory(openPath)
		d.Hide()
	})
	confirmBtn.Importance = widget.HighImportance

	form := widget.NewForm(
		widget.NewFormItem("目标目录:", container.NewBorder(nil, nil, nil, dstBtn, dstLabel)),
	)

	content := container.NewVBox(
		widget.NewLabelWithStyle("从保险库解密目录", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		form,
		vaultInfo,
		confirmBtn,
	)

	d = dialog.NewCustom("解密目录", "取消", content, w)
	d.Resize(fyne.NewSize(450, 200))
	d.Show()
}
