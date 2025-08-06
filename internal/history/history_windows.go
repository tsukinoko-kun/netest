package history

import (
	"os"
	"path/filepath"
)

func getHistoryDir() string {
	appdata := os.Getenv("APPDATA")
	return filepath.Join(appdata, "netest")
}
