package cmd

import (
	"fmt"
	"github.com/tsukinoko-kun/netest/internal/db"
	"github.com/tsukinoko-kun/netest/internal/server"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use: "serve [address]",
	RunE: func(cmd *cobra.Command, args []string) error {
		addr := ":4321"
		if len(args) > 0 {
			addr = args[0]
		}

		database, err := db.New()
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		defer database.Close()

		s, err := server.New(addr, database)
		if err != nil {
			return err
		}
		defer s.Stop(cmd.Context())
		time.Sleep(time.Second)
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
