package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tsukinoko-kun/netest/internal/db"
)

type dataResponse struct {
	TestResults []db.HistoryEntry `json:"test_results"`
}

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Print all test data as JSON array to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer database.Close()

		entries, err := db.RetrieveAll(database)
		if err != nil {
			return fmt.Errorf("failed to retrieve test results: %w", err)
		}

		response := dataResponse{TestResults: entries}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dataCmd)
}
