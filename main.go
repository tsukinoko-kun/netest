package main

import (
	"os"

	"github.com/tsukinoko-kun/netest/cmd"
	"github.com/tsukinoko-kun/netest/internal/db"
)

func main() {
	defer db.Close()
	if err := cmd.Execute(); err != nil {
		db.Close()
		os.Exit(1)
	}
}
