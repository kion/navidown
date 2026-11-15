package navidown

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// hangingServer answers after delay, or when the test ends.
func hangingServer(t *testing.T, delay time.Duration) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		select {
		case <-release:
		case <-time.After(delay):
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})
	return srv
}

func TestRemoteImageFetchGivesUpAfterTheTimeout(t *testing.T) {
	old := remoteFetchTimeout
	remoteFetchTimeout = 100 * time.Millisecond
	t.Cleanup(func() { remoteFetchTimeout = old })
	srv := hangingServer(t, 5*time.Second)

	start := time.Now()
	_, err := NewImageResolver(nil).Resolve(srv.URL+"/slow.png", "")
	if err == nil {
		t.Fatal("a server that never answers produced an image")
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("the fetch took %v, want it to stop at the timeout", took)
	}
}
