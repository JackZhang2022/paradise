APP_NAME := paradise
BUILD_DIR := build
DEBUG_SUFFIX := -debug
RELEASE_LDFLAGS := -ldflags="-X 'paradise/internal/logger.defaultLevelName=off'"
WIN_LDFLAGS := -ldflags="-H windowsgui -X 'paradise/internal/logger.defaultLevelName=off'"

.PHONY: all linux windows clean tidy run release release-linux release-windows

all: clean tidy linux windows
tidy:
	go mod tidy

linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-X 'paradise/internal/config.testMode=true'" -o $(BUILD_DIR)/$(APP_NAME)-linux$(DEBUG_SUFFIX).exe .
	@echo "✓ Linux 可执行文件已生成: $(BUILD_DIR)/$(APP_NAME)-linux$(DEBUG_SUFFIX).exe"

windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -ldflags="-H windowsgui -X 'paradise/internal/config.testMode=true'" -o $(BUILD_DIR)/$(APP_NAME)-windows$(DEBUG_SUFFIX).exe .
	@echo "✓ Windows 可执行文件已生成: $(BUILD_DIR)/$(APP_NAME)-windows$(DEBUG_SUFFIX).exe"

# 发布版本（不输出日志，不产生日志文件）
release: clean release-linux release-windows

release-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build $(RELEASE_LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64.exe .
	@echo "✓ Linux 发布版已生成: $(BUILD_DIR)/$(APP_NAME)-linux-amd64.exe"

release-windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build $(WIN_LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .
	@echo "✓ Windows 发布版已生成: $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe"

clean:
	rm -rf $(BUILD_DIR)
	@echo "✓ 已清理构建目录"

run:
	./$(BUILD_DIR)/$(APP_NAME)-linux$(DEBUG_SUFFIX).exe

publish:
	rm -f /run/media/geek/PARADISE/$(APP_NAME)-*
	cp $(BUILD_DIR)/$(APP_NAME)-*.exe /run/media/geek/PARADISE/