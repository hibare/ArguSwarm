// Package scout implements the scout agent functionality.
package scout

import (
	"github.com/spf13/cobra"
)

// ScoutCmd represents the scout command.
var ScoutCmd = &cobra.Command{
	Use:   "scout",
	Short: "Run the scout (Docker Swarm only)",
	Long:  `Run the scout agent for Docker Swarm. Scouts are not needed for Kubernetes deployments.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		agent, err := NewAgent()
		if err != nil {
			return err
		}
		return agent.Start()
	},
}
