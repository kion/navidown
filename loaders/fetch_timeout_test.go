package loaders

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/boolean-maybe/navidown/navidown"
)

func TestFollowedLinkFetchGivesUpAfterTheTimeout(t *testing.T) {
	old := remoteFetchTimeout
	remoteFetchTimeout = 100 * time.Millisecond
	t.Cleanup(func() { remoteFetchTimeout = old })
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	start := time.Now()
	_, err := (&FileHTTP{}).FetchContent(navidown.NavElement{URL: srv.URL + "/page.md"})
	if err == nil {
		t.Fatal("a server that never answers produced a page")
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("the fetch took %v, want it to stop at the timeout", took)
	}
}
