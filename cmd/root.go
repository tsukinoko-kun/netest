package cmd

import (
	"fmt"

	"github.com/tsukinoko-kun/netest/internal/networktest"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:               "netest",
	SilenceUsage:      true,
	DisableAutoGenTag: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		measurements, err := networktest.Run(cmd.Context())
		if err != nil {
			return fmt.Errorf("failed to run network test: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%+v\n", measurements)
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
