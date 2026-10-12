package gui

import (
	"image/color"
	"os"
	"paradise/internal/logger"
	"path/filepath"
	"runtime"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type paradiseTheme struct {
	fontOnce sync.Once
	fontRes  fyne.Resource
}

func (t *paradiseTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (t *paradiseTheme) Font(style fyne.TextStyle) fyne.Resource {
	t.fontOnce.Do(func() {
		t.fontRes = loadSystemFont()
	})
	if t.fontRes != nil {
		return t.fontRes
	}
	return theme.DefaultTheme().Font(style)
}

func (t *paradiseTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *paradiseTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func newTheme() fyne.Theme {
	return &paradiseTheme{}
}

func loadSystemFont() fyne.Resource {
	var paths []string
	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/share/fonts/windows/MiSans-Regular.otf",
			"/usr/share/fonts/windows/simkai.ttf",
			"/usr/share/fonts/wps-office/HTYYXSTJ.ttf",
			"/usr/share/fonts/wps-office/HYZYB5.ttf",
		}
	case "darwin":
		paths = []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/System/Library/Fonts/STHeiti Light.ttc",
			"/Library/Fonts/Arial Unicode.ttf",
		}
	case "windows":
		paths = []string{
			`C:\Windows\Fonts\msyh.ttc`,
			`C:\Windows\Fonts\simsun.ttc`,
			`C:\Windows\Fonts\simhei.ttf`,
		}
	}
	for _, p := range paths {
		ext := filepath.Ext(p)
		// Fyne 不支持 .ttc 格式，直接跳过避免文件读取开销
		if ext == ".ttc" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		logger.Info("加载系统字体", "path", p)
		return fyne.NewStaticResource(filepath.Base(p), data)
	}
	logger.Warn("未找到系统 CJK 字体，中文可能显示异常")
	return nil
}
