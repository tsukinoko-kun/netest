package cmd

import (
	"github.com/tsukinoko-kun/netest/internal/deamon"

	"github.com/spf13/cobra"
)

var (
	deamonCmd = &cobra.Command{
		Use:     "deamon",
		Aliases: []string{"service"},
		Short:   "Manage the deamon",
	}

	deamonInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install the deamon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				deamon.Addr = addr
			}

			deamon.Install()
		},
	}

	deamonUninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the deamon",
		Run: func(cmd *cobra.Command, args []string) {
			deamon.Uninstall()
		},
	}

	deamonStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the deamon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				deamon.Addr = addr
			}

			deamon.Start()
		},
	}

	deamonStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the deamon",
		Run: func(cmd *cobra.Command, args []string) {
			deamon.Stop()
		},
	}

	deamonStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Status of the deamon",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := deamon.StatusString()
			if err != nil {
				return err
			}
			cmd.Println(status)
			return nil
		},
	}

	deamonRunCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Short:  "Run the deamon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				deamon.Addr = addr
			}
			deamon.Run()
		},
	}
)

func init() {
	deamonCmd.AddCommand(deamonInstallCmd)
	deamonCmd.AddCommand(deamonUninstallCmd)
	deamonCmd.AddCommand(deamonStartCmd)
	deamonCmd.AddCommand(deamonStopCmd)
	deamonCmd.AddCommand(deamonStatusCmd)
	deamonCmd.AddCommand(deamonRunCmd)
	deamonInstallCmd.Flags().String("addr", "", "Listening address")
	deamonRunCmd.Flags().String("addr", "", "Listening address")
	deamonStartCmd.Flags().String("addr", "", "Listening address")
	rootCmd.AddCommand(deamonCmd)
}
