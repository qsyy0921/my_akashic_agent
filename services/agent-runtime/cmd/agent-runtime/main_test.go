package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultMediaAssetRootsDiscoverRepoFromRepoRoot(t *testing.T) {
	repoRoot := fakeAkashicRepo(t)

	roots := defaultMediaAssetRootsFrom([]string{repoRoot})

	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads"))
	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "generated_images"))
	assertContainsPath(t, roots, filepath.Join(repoRoot, "generated_images"))
	assertNotContainsPath(t, roots, filepath.Join(filepath.Dir(repoRoot), ".akashic-workspace", "uploads"))
}

func TestDefaultMediaAssetRootsDiscoverRepoFromServiceOrBinDir(t *testing.T) {
	repoRoot := fakeAkashicRepo(t)
	serviceDir := filepath.Join(repoRoot, "services", "agent-runtime")
	binDir := filepath.Join(repoRoot, ".tmp", "bin")
	if err := os.MkdirAll(serviceDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}

	roots := defaultMediaAssetRootsFrom([]string{serviceDir, binDir})

	assertContainsPath(t, roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads"))
	if got := countPath(roots, filepath.Join(repoRoot, ".akashic-workspace", "uploads")); got != 1 {
		t.Fatalf("expected upload root once, got %d in %#v", got, roots)
	}
}

func fakeAkashicRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, ".akashic-workspace", "uploads"),
		filepath.Join(root, "services", "agent-runtime"),
	} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project]\nname = \"akashic-test\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "services", "agent-runtime", "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func assertContainsPath(t *testing.T, paths []string, want string) {
	t.Helper()
	if countPath(paths, want) > 0 {
		return
	}
	t.Fatalf("expected %s in %#v", filepath.Clean(want), paths)
}

func assertNotContainsPath(t *testing.T, paths []string, want string) {
	t.Helper()
	if countPath(paths, want) == 0 {
		return
	}
	t.Fatalf("did not expect %s in %#v", filepath.Clean(want), paths)
}

func countPath(paths []string, want string) int {
	want = filepath.Clean(want)
	count := 0
	for _, path := range paths {
		if filepath.Clean(path) == want {
			count++
		}
	}
	return count
}
