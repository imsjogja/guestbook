package handler

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewTemplateRendererLoadsWebTemplates(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change to project root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDir) })

	if _, err := NewTemplateRenderer(); err != nil {
		t.Fatalf("load web templates: %v", err)
	}
}
