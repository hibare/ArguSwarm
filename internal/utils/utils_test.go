package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"length 1", 1},
		{"length 10", 10},
		{"length 100", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRandomString(tt.length)
			assert.Len(t, got, tt.length, "GetRandomString() length mismatch")

			// Verify all characters are from the charset
			for _, c := range got {
				assert.True(t, isInCharset(byte(c)), "GetRandomString() contains invalid character: %c", c)
			}
		})
	}
}

func isInCharset(c byte) bool {
	for _, valid := range charset {
		if byte(valid) == c {
			return true
		}
	}
	return false
}

func TestParallelExecute(t *testing.T) {
	tests := []struct {
		name    string
		items   []int
		worker  func(int) (string, error)
		want    []string
		wantErr []error
	}{
		{
			name:  "successful execution",
			items: []int{1, 2, 3},
			worker: func(i int) (string, error) {
				return string(rune('a' + i)), nil
			},
			want:    []string{"b", "c", "d"},
			wantErr: nil,
		},
		{
			name:  "with errors",
			items: []int{1, 2, 3},
			worker: func(i int) (string, error) {
				if i == 2 {
					return "", errors.New("test error")
				}
				return string(rune('a' + i)), nil
			},
			want:    []string{"b", "d"},
			wantErr: []error{errors.New("test error")},
		},
		{
			name:    "empty input",
			items:   []int{},
			worker:  func(_ int) (string, error) { return "", nil },
			want:    []string{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := ParallelExecute(tt.items, tt.worker)

			// Check results
			assert.Len(t, got, len(tt.want), "ParallelExecute() results length mismatch")

			// Sort results since order is not guaranteed
			gotMap := make(map[string]bool)
			for _, r := range got {
				gotMap[r] = true
			}
			for _, w := range tt.want {
				assert.True(t, gotMap[w], "ParallelExecute() missing expected result: %v", w)
			}

			// Check errors
			assert.Len(t, gotErr, len(tt.wantErr), "ParallelExecute() errors length mismatch")
			for i, err := range gotErr {
				assert.Equal(t, tt.wantErr[i].Error(), err.Error(), "ParallelExecute() error mismatch")
			}
		})
	}
}

func TestParallelExecuteConcurrency(t *testing.T) {
	// Test that all goroutines actually run in parallel
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}

	start := time.Now()
	results, _ := ParallelExecute(items, func(i int) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return i, nil
	})
	duration := time.Since(start)

	// If truly parallel, should take roughly 10ms, not 1000ms
	assert.Less(
		t,
		duration,
		100*time.Millisecond,
		"ParallelExecute() took too long, suggesting operations were not parallel",
	)

	// Verify all results were collected
	assert.Len(t, results, len(items), "ParallelExecute() missing results")
}
