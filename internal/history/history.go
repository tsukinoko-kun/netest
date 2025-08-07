package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type HistoryEntry[T any] struct {
	Value T
	Time  time.Time
}

func ExtractValue[T any](v HistoryEntry[T]) T {
	return v.Value
}

func ExtractTime[T any](v HistoryEntry[T]) time.Time {
	return v.Time
}

func Unpack[T any](v []HistoryEntry[T]) ([]T, time.Time) {
	result := make([]T, len(v))
	for i, e := range v {
		result[i] = e.Value
	}
	return result, v[len(v)/2].Time
}

var mut sync.RWMutex

func Track[T any](v T) error {
	mut.Lock()
	defer mut.Unlock()

	now := time.Now()
	entry := HistoryEntry[T]{
		v,
		now,
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

func Summarize[T any](join func(entries []HistoryEntry[T]) HistoryEntry[T]) error {
	mut.Lock()
	defer mut.Unlock()

	entries, err := Retrieve[T]()
	if err != nil {
		return fmt.Errorf("failed to retrieve entries: %w", err)
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastMonth := now.AddDate(0, -1, 0)
	threeMonthsAgo := now.AddDate(0, -3, 0)

	var result []HistoryEntry[T]
	var currentGroup []HistoryEntry[T]
	var currentGroupKey string

	for _, entry := range entries {
		// Skip entries older than 3 months - they should already be summarized
		if entry.Time.Before(threeMonthsAgo) {
			result = append(result, entry)
			continue
		}

		// Current day entries - don't touch
		if entry.Time.After(today) || entry.Time.Equal(today) {
			result = append(result, entry)
			continue
		}

		var groupKey string
		if entry.Time.After(lastMonth) {
			// Last month: group by hour
			groupKey = fmt.Sprintf("%d-%02d-%02d-%02d",
				entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), entry.Time.Hour())
		} else {
			// Older than last month: group by 6-hour periods
			sixHourPeriod := entry.Time.Hour() / 6
			groupKey = fmt.Sprintf("%d-%02d-%02d-%d",
				entry.Time.Year(), entry.Time.Month(), entry.Time.Day(), sixHourPeriod)
		}

		// If this is a new group, process the previous group
		if currentGroupKey != "" && currentGroupKey != groupKey {
			if len(currentGroup) > 1 {
				result = append(result, join(currentGroup))
			} else if len(currentGroup) == 1 {
				result = append(result, currentGroup[0])
			}
			currentGroup = nil
		}

		currentGroup = append(currentGroup, entry)
		currentGroupKey = groupKey
	}

	// Process the last group
	if len(currentGroup) > 1 {
		result = append(result, join(currentGroup))
	} else if len(currentGroup) == 1 {
		result = append(result, currentGroup[0])
	}

	// Store the summarized entries
	return store(result)
}

func getHistoryFile() string {
	if _, err := os.Stat(historyDir); os.IsNotExist(err) {
		_ = os.MkdirAll(historyDir, 0755)
	}
	return filepath.Join(historyDir, "history")
}

func Retrieve[T any]() ([]HistoryEntry[T], error) {
	mut.RLock()
	defer mut.RUnlock()

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

func store[T any](entries []HistoryEntry[T]) error {
	fileName := getHistoryFile()
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create history file: %w", err)
	}
	defer f.Close()

	jw := json.NewEncoder(f)
	for _, entry := range entries {
		if err := jw.Encode(entry); err != nil {
			return fmt.Errorf("failed to encode history entry: %w", err)
		}
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync history file: %w", err)
	}
	return nil
}
