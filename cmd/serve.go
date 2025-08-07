package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tsukinoko-kun/netest/internal/server"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use: "serve [address]",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := ":8111"
		if len(args) > 0 {
			addr = args[0]
		}

		s, err := server.New(addr)
		if err != nil {
			return err
		}
		defer s.Stop(cmd.Context())
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Listening on %s\n", s.ListeningAddr())
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
		<-ch
		return s.Stop(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
