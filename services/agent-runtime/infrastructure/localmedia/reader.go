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
	roots []string
}

func NewReader(roots []string) (*Reader, error) {
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
	return &Reader{roots: cleaned}, nil
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
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			if err == nil {
				return true
			}
			continue
		}
		if rel == "" || rel == "." {
			return true
		}
		if !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
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
