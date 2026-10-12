package main

import (
	"paradise/internal/config"
	"paradise/internal/gui"
	"paradise/internal/logger"
)

func main() {
	dataDir, err := config.DataDir()
	if err != nil {
		dataDir = "."
	}
	logger.Init(dataDir, logger.DefaultLevel())
	logger.Info("应用启动")
	gui.Run()
	logger.Info("应用退出")
}
