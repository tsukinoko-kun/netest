package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type HistoryEntry[T any] struct {
	Value T
	Time  time.Time
}

func Track[T any](v T) error {
	entry := HistoryEntry[T]{
		Value: v,
		Time:  time.Now(),
	}

	fileName := getHistoryFile()
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	jw := json.NewEncoder(f)
	if err := jw.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode history entry: %w", err)
	}

	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync history file: %w", err)
	}

	return nil
}

func getHistoryFile() string {
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		_ = os.MkdirAll(historyDir, 0755)
	}
	return filepath.Join(historyDir, "history")
}

func Retrieve[T any]() ([]HistoryEntry[T], error) {
	var entries []HistoryEntry[T]

	fileName := getHistoryFile()
	f, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		var entry HistoryEntry[T]
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal history entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
