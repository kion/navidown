package navidown

import (
	"fmt"
	"strings"
	"testing"
)

type cleanCountingRenderer struct {
	inner  Renderer
	cleans *int
}

func (r cleanCountingRenderer) Render(markdown string) (RenderResult, error) {
	res, err := r.inner.Render(markdown)
	if err != nil {
		return res, err
	}
	inner := res.Cleaner
	res.Cleaner = LineCleanerFunc(func(s string) string {
		*r.cleans++
		return inner.Clean(s)
	})
	return res, nil
}

// At a narrow width headings and links wrap, their markers split across rows,
// and every such element is matched by scanning all the rendered lines.
func TestMatchingCleansEachRenderedLineOnce(t *testing.T) {
	var md strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&md, "## Section %d about telemetry and firmware\n\n", i)
		fmt.Fprintf(&md, "Read [the long design notes number %d](https://example.com/%d) first.\n\n", i, i)
	}

	cleans := 0
	session := New(Options{Renderer: cleanCountingRenderer{
		inner:  NewANSIRenderer().WithWordWrap(11),
		cleans: &cleans,
	}})
	if err := session.SetMarkdown(md.String()); err != nil {
		t.Fatal(err)
	}

	lines := len(session.RenderedLines())
	if limit := 4 * lines; cleans > limit {
		t.Fatalf("%d rendered lines were cleaned %d times, want at most %d", lines, cleans, limit)
	}
}
