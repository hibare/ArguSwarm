// Package overseer provides the overseer command implementation
package overseer

import (
	"github.com/spf13/cobra"
)

// OverseerCmd represents the overseer command.
var OverseerCmd = &cobra.Command{
	Use:   "overseer",
	Short: "Run the overseer",
	Long:  ``,
	RunE: func(_ *cobra.Command, _ []string) error {
		overseer, err := NewOverseer()
		if err != nil {
			return err
		}
		return overseer.Start()
	},
}
