// Package node provides node identification and management functionality.
package node

import (
	"os"

	"github.com/hibare/ArguSwarm/internal/utils"
)

const (
	defaultNodeIDLength = 8
)

// hostnameFunc is used to get the hostname of the system.
var hostnameFunc = os.Hostname

// GetNodeID retrieves the unique identifier for the current node.
func GetNodeID() string {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		hostname, err := hostnameFunc()
		if err != nil {
			return "node-" + utils.GetRandomString(defaultNodeIDLength)
		}
		return hostname
	}
	return nodeID
}
