package cmd

import (
	"github.com/tsukinoko-kun/netest/internal/networktest"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "netest",
	RunE: func(cmd *cobra.Command, args []string) error {
		return networktest.Run()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
