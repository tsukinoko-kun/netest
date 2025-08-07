package cmd

import (
	"fmt"
	"github.com/tsukinoko-kun/netest/internal/db"
	"github.com/tsukinoko-kun/netest/internal/networktest"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "netest",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, err := db.New()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer database.Close()

		return networktest.Run(database)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
