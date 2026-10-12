# Paradise

Paradise 是一个本地加密保险库应用，使用 Go 语言和 Fyne GUI 框架构建，支持 Windows 和 Linux 平台。用户可以将敏感目录加密存入保险库，需要时解密查阅或修改，修改完成后可一键写回保险库并自动清理解密副本。

## 功能概览

| 功能 | 说明 |
|------|------|
| 🔒 加密目录 | 将明文目录加密存入保险库，原始目录自动删除 |
| 🔓 解密目录 | 将保险库中的加密目录解密还原到指定位置 |
| 📝 写回并清理 | 将解密后修改的目录重新加密写回保险库，并删除解密副本 |
| 🔑 修改密码 | 修改主密码，自动用新密钥重新加密整个保险库 |
| 📦 备份 | 将加密保险库打包为 tar.gz 备份 |
| ♻️ 恢复 | 从 tar.gz 备份恢复保险库 |

## 项目结构

```
paradise/
├── main.go                         # 入口
├── go.mod / go.sum                 # Go 模块定义
├── Makefile                        # 构建脚本
├── README.md
├── build/                          # 构建产物
│   ├── paradise-linux-amd64.exe
│   └── paradise-windows-amd64.exe
└── internal/
    ├── config/                     # 配置与认证
    │   ├── config.go               # 应用配置、数据目录管理
    │   └── auth.go                 # 密码设置、验证、修改密码
    ├── crypto/                     # 底层密码学操作（纯函数，无副作用）
    │   └── crypto.go               # 密钥派生、数据加解密、名称派生、文件加解密
    ├── vault/                      # 保险库业务逻辑
    │   ├── vault.go                # 包文档
    │   ├── encrypt.go              # 加密目录到保险库
    │   ├── decrypt.go              # 从保险库解密目录
    │   ├── writeback.go            # 写回并清理解密目录
    │   ├── rekey.go                # 重新加密整个保险库（修改密码时使用）
    │   ├── backup.go               # 创建/恢复备份
    │   ├── state.go                # 解密状态追踪（持久化记录）
    │   └── errors.go               # 领域错误定义
    ├── gui/                        # 图形用户界面（Fyne）
    │   ├── app.go                  # 应用入口，窗口管理
    │   ├── screen_login.go         # 登录界面
    │   ├── screen_setup.go         # 首次设置界面
    │   ├── screen_dashboard.go     # 主功能面板
    │   ├── screen_change_password.go # 修改密码界面
    │   ├── dialog_encrypt.go       # 加密目录对话框
    │   ├── dialog_decrypt.go       # 解密目录对话框
    │   ├── dialog_writeback.go     # 写回对话框
    │   ├── dialog_backup.go        # 备份对话框
    │   ├── dialog_restore.go       # 恢复备份对话框
    │   ├── operations.go           # 共享 UI 工具
    │   └── theme.go                # 中文字体主题
    └── logger/                     # 结构化日志
        ├── logger.go               # 基于 slog 的日志系统
        └── level.go                # 日志级别控制（支持构建时注入）
```

## 架构设计

### 分层架构

```
┌─────────────────────────────────┐
│              GUI                 │  用户交互、对话框、窗口管理
│         (internal/gui)           │
├─────────────────────────────────┤
│             Vault                │  保险库业务逻辑：加密/解密/写回/备份/重加密
│        (internal/vault)          │
├─────────────────────────────────┤
│     Config          Crypto       │  配置与认证管理      底层密码学操作
│  (internal/config)  (internal/crypto) │
├─────────────────────────────────┤
│            Logger                │  结构化日志
│        (internal/logger)         │
└─────────────────────────────────┘
```

各层职责清晰、单向依赖：GUI → Vault → Crypto/Config → Logger。

### 密码学设计

- **密钥派生**：使用 Argon2id 从用户密码 + 随机盐值派生 256-bit AES 密钥。盐值持久化在配置文件中，确保相同密码派生出相同密钥
- **数据加密**：AES-256-GCM（认证加密），每次加密生成随机 12 字节 nonce，确保同一文件多次加密产生不同密文
- **名称保护**：文件名和目录名使用 HMAC-SHA256 派生为 base64 短名称，防止暴露原始目录结构
- **文件元数据保护**：加密文件时，原始文件名嵌入密文中（`[2字节名称长度][名称字节][文件内容]`），解密时自动提取还原
- **目录元数据保护**：每个加密子目录包含 `.paradise_name` 文件，存储加密后的原始目录名
- **密码存储**：使用 bcrypt 哈希存储密码，不存储明文密码

### 状态管理

解密操作会记录到 `paradise_state.json` 文件中（AES-256-GCM 加密存储），用于跟踪已解密的目录与对应保险库的映射关系：

- 写回时自动查找匹配的解密记录
- 支持多个保险库子目录同时解密和独立写回
- 修改密码时状态文件也会用新密钥重新加密

### 日志系统

- 开发模式：JSON 格式日志输出到 `.paradise/paradise.log`
- 发布模式：通过 `-ldflags` 注入 `level=off`，完全不产生日志文件
- 环境变量 `PARADISE_LOG_LEVEL` 可覆盖默认级别

### 数据目录

应用运行时会在可执行文件同级创建 `.paradise/` 目录，存放：
- `paradise_config.json`：应用配置（保险库路径、盐值、密码哈希）
- `paradise_state.json`：加密状态记录
- `paradise.log`：运行日志（发布模式不产生）

## 构建与运行

```bash
# 安装依赖并构建所有平台
make all

# 仅构建 Linux
make linux

# 仅构建 Windows
make windows

# 构建发布版（无日志输出）
make release

# 运行
make run

# 清理构建产物
make clean
```

## TODO

* 解密目录自动打开目标目录
* 超时自动写回并清理