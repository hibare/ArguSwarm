// Package overseer provides the overseer command implementation
package overseer

import (
	httpin_integration "github.com/ggicci/httpin/integration"
	"github.com/go-chi/chi/v5"
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

func init() {
	// Register a directive named "path" to retrieve values from `chi.URLParam`,
	httpin_integration.UseGochiURLParam("path", chi.URLParam)
}
