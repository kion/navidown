package navidown

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// missingImageServer answers 404 and counts the requests.
func missingImageServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.NotFound(w, nil)
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

// A page is rendered again on every resize and refresh, on the UI goroutine:
// fetching a dead image each time would block it for as long as the failure takes.
func TestFailedImageIsNotFetchedAgain(t *testing.T) {
	srv, requests := missingImageServer(t)
	r := NewImageResolver(nil)

	for i := 0; i < 3; i++ {
		if _, err := r.Resolve(srv.URL+"/gone.png", ""); err == nil {
			t.Fatal("a missing image resolved")
		}
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("the missing image was fetched %d times, want 1", got)
	}
}

func TestRetryFailedFetchesAFailedImageAgain(t *testing.T) {
	srv, requests := missingImageServer(t)
	r := NewImageResolver(nil)
	url := srv.URL + "/gone.png"
	_, _ = r.Resolve(url, "")

	r.RetryFailed([]string{url, srv.URL + "/never-tried.png"}, "")

	if got := requests.Load(); got != 2 {
		t.Fatalf("after one retry the server saw %d requests, want 2", got)
	}
}

func TestClearCacheForgetsFailedImages(t *testing.T) {
	srv, requests := missingImageServer(t)
	r := NewImageResolver(nil)
	url := srv.URL + "/gone.png"
	_, _ = r.Resolve(url, "")

	r.ClearCache()
	_, _ = r.Resolve(url, "")

	if got := requests.Load(); got != 2 {
		t.Fatalf("after ClearCache the server saw %d requests, want 2", got)
	}
}

// countingFailingMmdc is a fake mmdc that counts its runs and always fails.
func countingFailingMmdc(t *testing.T) (script string, runs func() int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake mmdc is a shell script")
	}
	dir := t.TempDir()
	counter := filepath.Join(dir, "runs")
	script = filepath.Join(dir, "bad-mmdc")
	body := fmt.Sprintf("#!/bin/sh\necho x >> %q\nexit 1\n", counter)
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	return script, func() int {
		data, _ := os.ReadFile(counter)
		return strings.Count(string(data), "x")
	}
}

func failingMermaidRenderer(t *testing.T, script string, retry bool) *MermaidRenderer {
	t.Helper()
	r := NewMermaidRenderer(MermaidOptions{MmdcPath: script, CacheDir: t.TempDir(), RetryFailed: retry})
	if r == nil {
		t.Fatal("NewMermaidRenderer returned nil")
	}
	r.rasterizer = nil
	t.Cleanup(r.Close)
	return r
}

// A diagram with a syntax error fails after mmdc has started a browser, which takes seconds.
func TestFailedDiagramIsNotRenderedAgain(t *testing.T) {
	script, runs := countingFailingMmdc(t)
	source := "flowchart TD\n  A --> " + strconv.Itoa(os.Getpid()) + t.Name()

	if _, err := failingMermaidRenderer(t, script, false).RenderToFile(source); err == nil {
		t.Fatal("a failing mmdc produced a diagram")
	}
	first := runs()
	// every viewer has its own renderer
	if _, err := failingMermaidRenderer(t, script, false).RenderToFile(source); err == nil {
		t.Fatal("a failing mmdc produced a diagram")
	}

	if got := runs(); got != first {
		t.Fatalf("mmdc ran %d more times for a diagram that had already failed", got-first)
	}
}

func TestRetryFailedRendersAFailedDiagramAgain(t *testing.T) {
	script, runs := countingFailingMmdc(t)
	source := "flowchart TD\n  A --> " + strconv.Itoa(os.Getpid()) + t.Name()
	_, _ = failingMermaidRenderer(t, script, false).RenderToFile(source)
	first := runs()

	_, _ = failingMermaidRenderer(t, script, true).RenderToFile(source)

	if got := runs(); got == first {
		t.Fatal("a renderer told to retry did not run mmdc again")
	}
}
