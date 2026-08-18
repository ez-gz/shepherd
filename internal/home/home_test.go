package home

import (
	"path/filepath"
	"testing"

	"github.com/ez-gz/shepherd/internal/env"
)

// isolate points every path input at a temporary home so a test can never touch
// a developer's real installation.
func isolate(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("HOME", base)
	t.Setenv(env.Home, "")
	return base
}

func TestDirDefaultsToDotShepherd(t *testing.T) {
	base := isolate(t)
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if want := filepath.Join(base, DirName); dir != want {
		t.Fatalf("Dir = %q, want %q", dir, want)
	}
}

func TestDirHonorsOverride(t *testing.T) {
	isolate(t)
	custom := t.TempDir()
	t.Setenv(env.Home, custom)
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}
	if dir != custom {
		t.Fatalf("Dir = %q, want %q", dir, custom)
	}
}
