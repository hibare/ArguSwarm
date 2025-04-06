// Package utils provides common utility functions used throughout the application.
package utils

import (
	"crypto/rand"
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

// GetRandomString generates a cryptographically secure random string.
func GetRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// ParallelExecute executes multiple goroutines in parallel and collects their results.
// T is the type of the result, and R is the type of the input.
func ParallelExecute[T any, R any](items []R, worker func(R) (T, error)) ([]T, []error) {
	results := make(chan T, len(items))
	errors := make(chan error, len(items))

	// Launch goroutines for each item
	for _, item := range items {
		go func(r R) {
			result, err := worker(r)
			if err != nil {
				errors <- err
				return
			}
			results <- result
		}(item)
	}

	// Collect results and errors
	var collectedResults []T
	var collectedErrors []error

	for range items {
		select {
		case result := <-results:
			collectedResults = append(collectedResults, result)
		case err := <-errors:
			collectedErrors = append(collectedErrors, err)
		}
	}

	return collectedResults, collectedErrors
}
