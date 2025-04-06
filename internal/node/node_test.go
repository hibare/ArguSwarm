package node

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNodeID(t *testing.T) {
	// Save original hostname
	originalHostname, err := os.Hostname()
	require.NoError(t, err)

	tests := []struct {
		name     string
		nodeID   string
		expected string
	}{
		{
			name:     "with NODE_ID set",
			nodeID:   "test-node-123",
			expected: "test-node-123",
		},
		{
			name:     "with empty NODE_ID",
			nodeID:   "",
			expected: originalHostname,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv("NODE_ID", tt.nodeID)

			// Get node ID
			got := GetNodeID()

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetNodeID_ErrorHandling(t *testing.T) {
	// Set empty NODE_ID to force hostname fallback
	t.Setenv("NODE_ID", "")

	// Mock os.Hostname to return an error
	originalHostname := hostnameFunc
	defer func() {
		hostnameFunc = originalHostname
	}()

	hostnameFunc = func() (string, error) {
		return "", assert.AnError
	}

	// Get node ID
	got := GetNodeID()

	// Verify that we got a random string with the correct prefix
	assert.Contains(t, got, "node-")
	assert.Len(t, got, len("node-")+8) // 8 is the length of random string
}
