package agentruntime_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/kachofugetsu09/akashic-agent/services/agent-runtime"

var forbiddenImports = map[string]map[string]struct{}{
	"domain": {
		"api":            {},
		"app":            {},
		"infrastructure": {},
		"trigger":        {},
	},
	"app": {
		"api":            {},
		"infrastructure": {},
		"trigger":        {},
	},
	"infrastructure": {
		"api":     {},
		"trigger": {},
	},
	"trigger": {
		"domain":         {},
		"infrastructure": {},
	},
	"api": {
		"app":            {},
		"domain":         {},
		"infrastructure": {},
		"trigger":        {},
	},
	"types": {
		"api":            {},
		"app":            {},
		"domain":         {},
		"infrastructure": {},
		"trigger":        {},
	},
}

func TestLayerDependencyRules(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		layer := topLevelDir(root, path)
		forbidden, ok := forbiddenImports[layer]
		if !ok {
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			target := strings.Trim(imported.Path.Value, `"`)
			if !strings.HasPrefix(target, modulePath+"/") {
				continue
			}
			targetLayer := strings.Split(strings.TrimPrefix(target, modulePath+"/"), "/")[0]
			if _, blocked := forbidden[targetLayer]; blocked {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s must not import %s", filepath.ToSlash(rel), target)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func topLevelDir(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

