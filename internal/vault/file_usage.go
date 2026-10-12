package vault

import (
	"os"
	"path/filepath"
	"strconv"

	"paradise/internal/logger"
)

// CheckFilesInUse 检查目录下是否有文件被其他进程占用，返回占用的文件列表
func CheckFilesInUse(dir string) []string {
	var busy []string
	absDir, _ := filepath.Abs(dir)

	pids := getPIDs()

	_ = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		for _, pid := range pids {
			if pid == os.Getpid() {
				continue
			}
			if hasFileOpen(pid, path) {
				logger.Debug("文件被占用", "file", path, "pid", pid)
				busy = append(busy, path)
				break
			}
		}
		return nil
	})

	return busy
}

func getPIDs() []int {
	procDir, err := os.Open("/proc")
	if err != nil {
		return nil
	}
	defer procDir.Close()

	names, err := procDir.Readdirnames(-1)
	if err != nil {
		return nil
	}

	var pids []int
	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

func hasFileOpen(pid int, path string) bool {
	fdDir := filepath.Join("/proc", strconv.Itoa(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		link, err := os.Readlink(filepath.Join(fdDir, entry.Name()))
		if err != nil {
			continue
		}
		if link == path {
			return true
		}
	}
	return false
}
