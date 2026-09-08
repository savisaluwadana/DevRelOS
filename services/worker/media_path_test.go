package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfinedPathAllowsPathsInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "clips"), 0o755); err != nil {
		t.Fatal(err)
	}
	// On macOS t.TempDir() sits under /var, which is itself a symlink to
	// /private/var, so compare against the resolved root.
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	for _, candidate := range []string{"clips/a.mp4", "a.mp4", filepath.Join(root, "clips", "b.mp4")} {
		got, err := confinedPath(root, candidate)
		if err != nil {
			t.Fatalf("confinedPath(%q) rejected a path inside the root: %v", candidate, err)
		}
		if !strings.HasPrefix(got, resolvedRoot) {
			t.Fatalf("confinedPath(%q) = %q, expected it under %q", candidate, got, resolvedRoot)
		}
	}
}

func TestConfinedPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	for _, candidate := range []string{"../escape.mp4", "clips/../../escape.mp4", "/etc/passwd"} {
		if _, err := confinedPath(root, candidate); err == nil {
			t.Fatalf("confinedPath(%q) allowed a path outside the root", candidate)
		}
	}
}

func TestConfinedPathRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "media")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// A symlink living inside the media root but pointing out of it. Lexical
	// cleaning sees "<root>/leak/secret.mp4" and reports no traversal.
	if err := os.Symlink(outside, filepath.Join(root, "leak")); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}

	if _, err := confinedPath(root, "leak/secret.mp4"); err == nil {
		t.Fatal("confinedPath followed a symlink out of the media root")
	}

	// A symlinked root itself must still be usable.
	linkedRoot := filepath.Join(base, "media-link")
	if err := os.Symlink(root, linkedRoot); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	if _, err := confinedPath(linkedRoot, "clip.mp4"); err != nil {
		t.Fatalf("confinedPath rejected a legitimate path under a symlinked root: %v", err)
	}
}
