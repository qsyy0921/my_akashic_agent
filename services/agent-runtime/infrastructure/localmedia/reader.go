package localmedia

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	outport "github.com/kachofugetsu09/akashic-agent/services/agent-runtime/app/port/out"
	"github.com/kachofugetsu09/akashic-agent/services/agent-runtime/domain/model"
)

type Reader struct {
	roots                []string
	allowDiscoveredRoots bool
}

func NewReader(roots []string) (*Reader, error) {
	return NewReaderWithDiscoveredRoots(roots, false)
}

func NewReaderWithDiscoveredRoots(roots []string, allowDiscoveredRoots bool) (*Reader, error) {
	cleaned := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
			abs = evaluated
		}
		cleaned = append(cleaned, filepath.Clean(abs))
	}
	return &Reader{roots: cleaned, allowDiscoveredRoots: allowDiscoveredRoots}, nil
}

func (r *Reader) OpenMediaAssetContent(ctx context.Context, asset model.MediaAsset) (outport.MediaAssetContent, error) {
	if err := ctx.Err(); err != nil {
		return outport.MediaAssetContent{}, err
	}
	if r == nil || len(r.roots) == 0 {
		return outport.MediaAssetContent{}, outport.ErrMediaAssetContentDisabled
	}
	path, err := localPath(asset)
	if err != nil {
		return outport.MediaAssetContent{}, err
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return outport.MediaAssetContent{}, err
	}
	if evaluated, err := filepath.EvalSymlinks(resolved); err == nil {
		resolved = evaluated
	}
	resolved = filepath.Clean(resolved)
	if !r.allowed(resolved) {
		return outport.MediaAssetContent{}, outport.ErrMediaAssetContentForbidden
	}
	file, err := os.Open(resolved)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return outport.MediaAssetContent{}, outport.ErrMediaAssetContentUnavailable
		}
		return outport.MediaAssetContent{}, err
	}
	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return outport.MediaAssetContent{}, err
	}
	if stat.IsDir() {
		file.Close()
		return outport.MediaAssetContent{}, outport.ErrMediaAssetContentUnavailable
	}
	mimeType := strings.TrimSpace(asset.MimeType)
	if mimeType == "" {
		mimeType = detectMimeType(file, resolved)
	}
	name := strings.TrimSpace(asset.Name)
	if name == "" {
		name = filepath.Base(resolved)
	}
	return outport.MediaAssetContent{
		Name:      name,
		MimeType:  mimeType,
		SizeBytes: stat.Size(),
		Body:      file,
	}, nil
}

func (r *Reader) allowed(path string) bool {
	for _, root := range r.roots {
		if pathInsideRoot(root, path) {
			return true
		}
	}
	if !r.allowDiscoveredRoots {
		return false
	}
	for _, root := range discoveredAkashicMediaRoots(path) {
		if pathInsideRoot(root, path) {
			return true
		}
	}
	return false
}

func pathInsideRoot(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "" || rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) &&
		rel != ".." &&
		!filepath.IsAbs(rel)
}

func discoveredAkashicMediaRoots(path string) []string {
	current := filepath.Dir(path)
	roots := make([]string, 0, 3)
	seen := make(map[string]struct{})
	for {
		if isAkashicRepoRoot(current) {
			for _, candidate := range []string{
				filepath.Join(current, ".akashic-workspace", "uploads"),
				filepath.Join(current, ".akashic-workspace", "generated_images"),
				filepath.Join(current, "generated_images"),
			} {
				cleaned := filepath.Clean(candidate)
				key := strings.ToLower(cleaned)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				roots = append(roots, cleaned)
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return roots
}

func isAkashicRepoRoot(path string) bool {
	if stat, err := os.Stat(filepath.Join(path, ".akashic-workspace")); err == nil && stat.IsDir() {
		return true
	}
	if _, err := os.Stat(filepath.Join(path, "pyproject.toml")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(path, "services", "agent-runtime", "go.mod")); err != nil {
		return false
	}
	return true
}

func localPath(asset model.MediaAsset) (string, error) {
	candidates := []string{
		strings.TrimSpace(asset.URL),
		strings.TrimSpace(asset.Metadata["local_path"]),
		strings.TrimSpace(asset.Metadata["file_path"]),
		strings.TrimSpace(asset.Metadata["path"]),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		path, err := parseLocalPath(candidate)
		if err != nil {
			return "", err
		}
		if path != "" {
			return path, nil
		}
	}
	return "", outport.ErrMediaAssetContentUnavailable
}

func parseLocalPath(raw string) (string, error) {
	if filepath.VolumeName(raw) != "" || strings.HasPrefix(raw, `\\`) {
		return raw, nil
	}
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Scheme != "" {
		if parsed.Scheme != "file" {
			return "", outport.ErrMediaAssetContentUnavailable
		}
		path, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return "", err
		}
		if parsed.Host != "" {
			path = `\\` + parsed.Host + filepath.FromSlash(path)
		}
		if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		return filepath.FromSlash(path), nil
	}
	if strings.Contains(raw, "://") {
		return "", outport.ErrMediaAssetContentUnavailable
	}
	return raw, nil
}

func detectMimeType(file *os.File, path string) string {
	if byExt := mime.TypeByExtension(filepath.Ext(path)); byExt != "" {
		return byExt
	}
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	_, _ = file.Seek(0, 0)
	if n > 0 {
		return http.DetectContentType(buffer[:n])
	}
	return "application/octet-stream"
}
