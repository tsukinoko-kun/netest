package history

import (
	"os"
	"path/filepath"
)

func getHistoryDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, ".netest")
}
