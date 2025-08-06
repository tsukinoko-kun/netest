package cmd

import (
	"github.com/tsukinoko-kun/netest/internal/daemon"

	"github.com/spf13/cobra"
)

var (
	daemonCmd = &cobra.Command{
		Use:     "daemon",
		Aliases: []string{"service"},
		Short:   "Manage the daemon",
	}

	daemonInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				daemon.Addr = addr
			}

			daemon.Install()
		},
	}

	daemonUninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			daemon.Uninstall()
		},
	}

	daemonStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				daemon.Addr = addr
			}

			daemon.Start()
		},
	}

	daemonStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			daemon.Stop()
		},
	}

	daemonStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Status of the daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := daemon.StatusString()
			if err != nil {
				return err
			}
			cmd.Println(status)
			return nil
		},
	}

	daemonRunCmd = &cobra.Command{
		Use:    "run",
		Hidden: true,
		Short:  "Run the daemon",
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("addr") {
				addr, _ := cmd.Flags().GetString("addr")
				daemon.Addr = addr
			}
			daemon.Run()
		},
	}
)

func init() {
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonRunCmd)
	daemonInstallCmd.Flags().String("addr", "", "Listening address")
	daemonRunCmd.Flags().String("addr", "", "Listening address")
	daemonStartCmd.Flags().String("addr", "", "Listening address")
	rootCmd.AddCommand(daemonCmd)
}
