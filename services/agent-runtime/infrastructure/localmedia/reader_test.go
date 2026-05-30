package localmedia_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/infrastructure/localmedia"
)

func TestReaderServesFileInsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "qq-image.txt")
	if err := writeFile(path, "hello asset"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReader([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	content, err := reader.OpenMediaAssetContent(context.Background(), testAsset(path))
	if err != nil {
		t.Fatalf("open content: %v", err)
	}
	defer content.Body.Close()
	body, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello asset" {
		t.Fatalf("unexpected body: %s", string(body))
	}
	if content.Name != "qq-image.txt" {
		t.Fatalf("unexpected name: %s", content.Name)
	}
}

func TestReaderRejectsFileOutsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	path := filepath.Join(outside, "secret.txt")
	if err := writeFile(path, "secret"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReader([]string{root})
	if err != nil {
		t.Fatal(err)
	}

	_, err = reader.OpenMediaAssetContent(context.Background(), testAsset(path))
	if !errors.Is(err, outport.ErrMediaAssetContentForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestReaderCanDiscoverAkashicWorkspaceUploadRoot(t *testing.T) {
	root := fakeAkashicRepo(t)
	path := filepath.Join(root, ".akashic-workspace", "uploads", "qq-image.jpg")
	if err := writeFile(path, "image bytes"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReaderWithDiscoveredRoots([]string{t.TempDir()}, true)
	if err != nil {
		t.Fatal(err)
	}

	content, err := reader.OpenMediaAssetContent(context.Background(), testAsset(path))
	if err != nil {
		t.Fatalf("open content: %v", err)
	}
	defer content.Body.Close()
	body, err := io.ReadAll(content.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "image bytes" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestReaderDoesNotDiscoverAkashicRootsWhenDisabled(t *testing.T) {
	root := fakeAkashicRepo(t)
	path := filepath.Join(root, ".akashic-workspace", "uploads", "qq-image.jpg")
	if err := writeFile(path, "image bytes"); err != nil {
		t.Fatal(err)
	}
	reader, err := localmedia.NewReaderWithDiscoveredRoots([]string{t.TempDir()}, false)
	if err != nil {
		t.Fatal(err)
	}

	_, err = reader.OpenMediaAssetContent(context.Background(), testAsset(path))
	if !errors.Is(err, outport.ErrMediaAssetContentForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func testAsset(path string) model.MediaAsset {
	asset, err := model.NewMediaAsset(model.ChannelRef{
		Kind:             "qq",
		AccountID:        "1049511700",
		ConversationID:   "27234224",
		ConversationType: "group",
	}, model.MediaAssetSpec{
		AssetID:         "asset:qq:image:27234224:msg-1:1",
		SourceMessageID: "msg-1",
		SenderID:        "2948770636",
		Kind:            model.MediaAssetImage,
		URL:             path,
		Name:            filepath.Base(path),
	}, time.Now().UTC())
	if err != nil {
		panic(err)
	}
	return asset
}

func writeFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
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
