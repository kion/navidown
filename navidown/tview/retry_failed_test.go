package tview

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	nav "github.com/boolean-maybe/navidown/navidown"
)

// A viewer renders on the UI goroutine, so it must not repeat a failure;
// PreResolveMarkdown runs off it, and is where a failed image gets another chance.
func TestPreResolveMarkdownRetriesAFailedImageAndAViewerDoesNot(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.NotFound(w, nil)
	}))
	defer srv.Close()
	md := "![gone](" + srv.URL + "/gone.png)\n"
	mgr := NewImageManager(nav.NewImageResolver(nil), 8, 16)

	mgr.PreResolveMarkdown(md, "", 80, nil, nil)
	if got := requests.Load(); got != 1 {
		t.Fatalf("the first pass made %d requests, want 1", got)
	}

	viewer := nav.New(nav.Options{ImagePostProcessor: NewKittyImageProcessor(mgr)})
	if err := viewer.SetMarkdown(md); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("a viewer fetched the failed image again: %d requests", got)
	}

	mgr.PreResolveMarkdown(md, "", 80, nil, nil)
	if got := requests.Load(); got != 2 {
		t.Fatalf("the second pass left the failed image alone: %d requests, want 2", got)
	}
}

func TestPreResolveMarkdownRetriesAFailedDiagramAndAViewerDoesNot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake mmdc is a shell script")
	}
	dir := t.TempDir()
	counter := filepath.Join(dir, "runs")
	script := filepath.Join(dir, "bad-mmdc")
	if err := os.WriteFile(script, []byte(fmt.Sprintf("#!/bin/sh\necho x >> %q\nexit 1\n", counter)), 0755); err != nil {
		t.Fatal(err)
	}
	runs := func() int {
		data, _ := os.ReadFile(counter)
		return strings.Count(string(data), "x")
	}
	opts := &nav.MermaidOptions{MmdcPath: script, CacheDir: t.TempDir()}
	md := "```mermaid\nflowchart TD\n  A --> " + t.Name() + "\n```\n"
	mgr := NewImageManager(nav.NewImageResolver(nil), 8, 16)

	mgr.PreResolveMarkdown(md, "", 80, opts, nil)
	first := runs()
	if first == 0 {
		t.Fatal("the first pass never ran mmdc")
	}

	viewer := nav.New(nav.Options{MermaidOptions: opts})
	defer viewer.Close()
	if err := viewer.SetMarkdown(md); err != nil {
		t.Fatal(err)
	}
	if got := runs(); got != first {
		t.Fatalf("a viewer ran mmdc %d more times for a diagram that had already failed", got-first)
	}

	mgr.PreResolveMarkdown(md, "", 80, opts, nil)
	if got := runs(); got == first {
		t.Fatal("the second pass did not try the failed diagram again")
	}
}
