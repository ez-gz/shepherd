// Package home resolves the single directory that holds every Shepherd file:
// settings, durable application state, workstream artifacts, and the agent
// instructions a manager session reads.
package home

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ez-gz/shepherd/internal/env"
)

// DirName is created under the user's home directory by default.
const DirName = ".shepherd"

// Dir resolves the Shepherd home directory without creating it.
func Dir() (string, error) {
	if value := env.Value(env.Home); value != "" {
		path, err := filepath.Abs(value)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", env.Home, err)
		}
		return path, nil
	}
	base, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(base, DirName), nil
}

// Ensure resolves the Shepherd home directory and creates it privately.
func Ensure() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create shepherd home %q: %w", dir, err)
	}
	return dir, nil
}

// Path joins elements onto the Shepherd home directory.
func Path(elements ...string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{dir}, elements...)...), nil
}
