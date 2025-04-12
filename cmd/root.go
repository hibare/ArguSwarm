// Package cmd provides the command-line interface for the application.
package cmd

import (
	"context"
	"os"

	"github.com/hibare/ArguSwarm/cmd/overseer"
	"github.com/hibare/ArguSwarm/cmd/scout"
	"github.com/hibare/ArguSwarm/internal/config"
	"github.com/hibare/ArguSwarm/internal/version"
	commonLogger "github.com/hibare/GoCommon/v2/pkg/logger"
	"github.com/spf13/cobra"
)

// rootCmd is the root command for the CLI application.
var rootCmd = &cobra.Command{
	Use:     "arguswarm",
	Short:   "ArguSwarm is a tool for managing Docker containers",
	Long:    ``,
	Version: version.CurrentVersion,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute runs the root command and handles any errors.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	ctx := context.Background()

	rootCmd.SetContext(ctx)
	rootCmd.AddCommand(overseer.OverseerCmd)
	rootCmd.AddCommand(scout.ScoutCmd)

	cobra.OnInitialize(commonLogger.InitDefaultLogger, config.Load)
}
