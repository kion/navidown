package loaders

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/boolean-maybe/navidown/navidown"
)

// FileHTTP implements navidown.ContentProvider to fetch content from HTTP(S) URLs and local files.
type FileHTTP struct {
	// SearchRoots are extra directories to try when resolving relative file links.
	SearchRoots []string

	// Client is used for HTTP(S) requests; if nil, a client with a 20 s timeout is used.
	Client *http.Client
}

func (f *FileHTTP) FetchContent(elem navidown.NavElement) (string, error) {
	url := elem.URL
	if url == "" {
		return "", nil
	}

	resolvedPath, err := navidown.ResolveMarkdownPath(url, elem.SourceFilePath, f.SearchRoots)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %q: %w", url, err)
	}

	if strings.HasPrefix(resolvedPath, "http://") || strings.HasPrefix(resolvedPath, "https://") {
		return f.fetchFromWeb(resolvedPath)
	}
	return f.fetchFromLocal(resolvedPath)
}

// A followed link is fetched on the UI goroutine,
// so a server that never answers must not hold it forever.
var remoteFetchTimeout = 20 * time.Second

func (f *FileHTTP) fetchFromWeb(url string) (content string, err error) {
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: remoteFetchTimeout}
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func (f *FileHTTP) fetchFromLocal(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read local file: %w", err)
	}
	return string(content), nil
}
