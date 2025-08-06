package history

import (
	"os"
	"path/filepath"
)

func getHistoryDir() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, "Library", "Application Support", "netest")
}
