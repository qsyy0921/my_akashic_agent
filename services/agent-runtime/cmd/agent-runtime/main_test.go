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

func TestOneBotEndpointsFromEnvReadsMultiEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_HTTP_BASE_URLS", "qq_1049511700=http://127.0.0.1:3001, qq_2365524513=http://127.0.0.1:3002")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKENS", "qq_1049511700=token-a,qq_2365524513=token-b")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "default-token")

	endpoints := onebotEndpointsFromEnv()

	if len(endpoints) != 2 {
		t.Fatalf("expected two endpoints, got %#v", endpoints)
	}
	if got := endpoints["qq_1049511700"].BaseURL; got != "http://127.0.0.1:3001" {
		t.Fatalf("unexpected endpoint A base url: %q", got)
	}
	if got := endpoints["qq_1049511700"].AccessToken; got != "token-a" {
		t.Fatalf("unexpected endpoint A token: %q", got)
	}
	if got := endpoints["qq_2365524513"].BaseURL; got != "http://127.0.0.1:3002" {
		t.Fatalf("unexpected endpoint B base url: %q", got)
	}
	if got := endpoints["qq_2365524513"].AccessToken; got != "token-b" {
		t.Fatalf("unexpected endpoint B token: %q", got)
	}
}

func TestOneBotEndpointsFromEnvReadsSingleEndpointConfig(t *testing.T) {
	t.Setenv("AKASHIC_ONEBOT_HTTP_BASE_URL", "http://127.0.0.1:3001")
	t.Setenv("AKASHIC_ONEBOT_CHANNELS", "qq,qq_2365524513")
	t.Setenv("AKASHIC_ONEBOT_ACCESS_TOKEN", "shared-token")

	endpoints := onebotEndpointsFromEnv()

	if len(endpoints) != 2 {
		t.Fatalf("expected two channel aliases, got %#v", endpoints)
	}
	for _, channel := range []string{"qq", "qq_2365524513"} {
		endpoint, ok := endpoints[channel]
		if !ok {
			t.Fatalf("missing endpoint for %s: %#v", channel, endpoints)
		}
		if endpoint.BaseURL != "http://127.0.0.1:3001" || endpoint.AccessToken != "shared-token" {
			t.Fatalf("unexpected endpoint for %s: %#v", channel, endpoint)
		}
	}
}

func TestParseKeyValueCSVSkipsMalformedEntries(t *testing.T) {
	got := parseKeyValueCSV("a=1, malformed, b = 2, =missing, c= ")

	if len(got) != 2 || got["a"] != "1" || got["b"] != "2" {
		t.Fatalf("unexpected parse result: %#v", got)
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
