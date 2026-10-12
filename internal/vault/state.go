package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"paradise/internal/config"
	"paradise/internal/crypto"
	"paradise/internal/logger"
)

// DecryptRecord 记录一次解密操作
type DecryptRecord struct {
	SourceDir     string    `json:"source_dir"`
	DecryptedDir  string    `json:"decrypted_dir"`
	DecryptedTime time.Time `json:"decrypted_time"`
	Salt          string    `json:"salt"`
}

// State 保险库状态
type State struct {
	Records        []DecryptRecord `json:"records"`
	LastEncryptDir string          `json:"last_encrypt_dir,omitempty"`
	LastDecryptDir string          `json:"last_decrypt_dir,omitempty"`
}

const stateFileName = "paradise_state.json"

func stateFilePath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFileName), nil
}

// LoadState 加载并解密状态
func LoadState(key []byte) (*State, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("状态文件不存在，返回空状态", "path", path)
			return &State{}, nil
		}
		logger.Error("读取状态文件失败", "path", path, "error", err)
		return nil, err
	}
	plaintext, err := crypto.Decrypt(data, key)
	if err != nil {
		logger.Warn("状态文件解密失败，返回空状态", "path", path)
		return &State{}, nil
	}
	var state State
	if err := json.Unmarshal(plaintext, &state); err != nil {
		logger.Warn("状态文件解析失败，返回空状态", "path", path, "error", err)
		return &State{}, nil
	}
	logger.Debug("状态加载成功", "records", len(state.Records))
	return &state, nil
}

// SaveState 加密并保存状态
func SaveState(state *State, key []byte) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		logger.Error("序列化状态失败", "error", err)
		return fmt.Errorf("序列化状态失败: %w", err)
	}
	encrypted, err := crypto.Encrypt(data, key)
	if err != nil {
		logger.Error("加密状态失败", "error", err)
		return fmt.Errorf("加密状态失败: %w", err)
	}
	if err := os.WriteFile(path, encrypted, 0600); err != nil {
		logger.Error("写入状态文件失败", "path", path, "error", err)
		return err
	}
	logger.Debug("状态保存成功", "records", len(state.Records))
	return nil
}

// AddRecord 添加一条解密记录，若相同 source+decrypted 已存在则更新
func AddRecord(record DecryptRecord, key []byte) error {
	state, err := LoadState(key)
	if err != nil {
		return err
	}
	for i, r := range state.Records {
		if r.SourceDir == record.SourceDir && r.DecryptedDir == record.DecryptedDir {
			state.Records[i].DecryptedTime = record.DecryptedTime
			state.Records[i].Salt = record.Salt
			logger.Info("更新解密记录", "source", record.SourceDir, "decrypted", record.DecryptedDir)
			return SaveState(state, key)
		}
	}
	state.Records = append(state.Records, record)
	logger.Info("添加解密记录", "source", record.SourceDir, "decrypted", record.DecryptedDir)
	return SaveState(state, key)
}

// RemoveRecord 移除一条解密记录
func RemoveRecord(sourceDir string, key []byte) error {
	state, err := LoadState(key)
	if err != nil {
		return err
	}
	filtered := make([]DecryptRecord, 0, len(state.Records))
	for _, r := range state.Records {
		if r.SourceDir != sourceDir {
			filtered = append(filtered, r)
		}
	}
	state.Records = filtered
	logger.Info("移除解密记录", "source", sourceDir)
	return SaveState(state, key)
}

// ReEncryptState 用新密钥重新加密状态文件
func ReEncryptState(oldKey, newKey []byte) error {
	state, err := LoadState(oldKey)
	if err != nil {
		logger.Error("加载状态失败", "error", err)
		return fmt.Errorf("加载状态失败: %w", err)
	}
	logger.Info("重新加密状态文件", "records", len(state.Records))
	return SaveState(state, newKey)
}

// FindBySource 按源目录查找记录
func FindBySource(sourceDir string, key []byte) (*DecryptRecord, error) {
	state, err := LoadState(key)
	if err != nil {
		return nil, err
	}
	for _, r := range state.Records {
		if r.SourceDir == sourceDir {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("未找到源目录 %s 的解密记录", sourceDir)
}

// FindByDecrypted 按解密目录查找记录
func FindByDecrypted(decryptedDir string, key []byte) (*DecryptRecord, error) {
	state, err := LoadState(key)
	if err != nil {
		return nil, err
	}
	for _, r := range state.Records {
		if r.DecryptedDir == decryptedDir {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("未找到解密目录 %s 的记录", decryptedDir)
}

// GetAllRecords 获取所有记录
func GetAllRecords(key []byte) ([]DecryptRecord, error) {
	state, err := LoadState(key)
	if err != nil {
		return nil, err
	}
	return state.Records, nil
}

// SetLastEncryptDir 保存上次加密的源目录
func SetLastEncryptDir(dir string, key []byte) error {
	state, err := LoadState(key)
	if err != nil {
		return err
	}
	state.LastEncryptDir = dir
	return SaveState(state, key)
}

// GetLastEncryptDir 获取上次加密的源目录
func GetLastEncryptDir(key []byte) string {
	state, err := LoadState(key)
	if err != nil {
		return ""
	}
	return state.LastEncryptDir
}

// SetLastDecryptDir 保存上次解密的目标目录
func SetLastDecryptDir(dir string, key []byte) error {
	state, err := LoadState(key)
	if err != nil {
		return err
	}
	state.LastDecryptDir = dir
	return SaveState(state, key)
}

// GetLastDecryptDir 获取上次解密的目标目录
func GetLastDecryptDir(key []byte) string {
	state, err := LoadState(key)
	if err != nil {
		return ""
	}
	return state.LastDecryptDir
}
