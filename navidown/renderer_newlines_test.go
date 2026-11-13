package navidown

import (
	"slices"
	"strings"
	"testing"
)

func cleanLines(res RenderResult) []string {
	out := make([]string, 0, len(res.Lines))
	for _, l := range res.Lines {
		out = append(out, strings.TrimRight(res.Cleaner.Clean(l), " "))
	}
	for len(out) > 0 && out[0] == "" {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func renderLines(t *testing.T, r *ANSIStyleRenderer, md string) []string {
	t.Helper()
	res, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	return cleanLines(res)
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("lines mismatch\n got: %q\nwant: %q", got, want)
	}
}

const letteredOptions = "The fix:\n" +
	"A. First option, wrapped by hand\n" +
	"   onto an indented line.\n" +
	"B. Second option."

func TestRender_JoinsSoftBreaksByDefault(t *testing.T) {
	got := renderLines(t, NewANSIRendererWithStyle("dark").WithWordWrap(100), letteredOptions)
	assertLines(t, got, []string{
		"The fix: A. First option, wrapped by hand onto an indented line. B. Second option.",
	})
}

func TestRender_PreservedNewLines(t *testing.T) {
	tests := []struct {
		name  string
		md    string
		width int
		want  []string
	}{
		{
			name:  "lettered options keep breaks and indent",
			md:    letteredOptions,
			width: 80,
			want: []string{
				"The fix:",
				"A. First option, wrapped by hand",
				"   onto an indented line.",
				"B. Second option.",
			},
		},
		{
			name:  "wrapped rows keep their line's indent",
			md:    "A. alpha beta gamma\n   delta epsilon zeta eta theta",
			width: 16,
			want: []string{
				"A. alpha beta",
				"gamma",
				"   delta epsilon",
				"   zeta eta",
				"   theta",
			},
		},
		{
			name:  "indent wider than the pane is dropped",
			md:    "one\n          two three",
			width: 8,
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "tab indent counts to the next tab stop",
			md:    "A. x\n\ty",
			width: 80,
			want:  []string{"A. x", "    y"},
		},
		{
			name:  "bullet continuation lands under the item text",
			md:    "- item one\n  continued here\n- item two",
			width: 80,
			want:  []string{"• item one", "  continued here", "• item two"},
		},
		{
			name:  "wide bullet marker still aligns under the text",
			md:    "-   item\n    continued",
			width: 80,
			want:  []string{"• item", "  continued"},
		},
		{
			name:  "lazy continuation gets no indent",
			md:    "- item\nlazy",
			width: 80,
			want:  []string{"• item", "lazy"},
		},
		{
			name:  "ordered continuation",
			md:    "1. first\n   more\n2. second",
			width: 80,
			want:  []string{"1. first", "   more", "2. second"},
		},
		{
			name:  "two-digit ordered marker",
			md:    "10. ten\n    more",
			width: 80,
			want:  []string{"10. ten", "    more"},
		},
		{
			name:  "repeated ordered marker is renumbered",
			md:    "1. first\n1. second\n   more",
			width: 80,
			want:  []string{"1. first", "2. second", "   more"},
		},
		{
			name:  "task continuation lands under the task text",
			md:    "- [ ] todo\n      more",
			width: 80,
			want:  []string{"[ ] todo", "    more"},
		},
		{
			name:  "nested list continuation",
			md:    "- outer\n  - inner\n    more",
			width: 80,
			want:  []string{"• outer", "  • inner", "    more"},
		},
		{
			name:  "blockquote with marker on the continuation",
			md:    "> quote line\n>   indented more\n> back",
			width: 80,
			want:  []string{"│ quote line", "│   indented more", "│ back"},
		},
		{
			name:  "blockquote lazy continuation",
			md:    "> quote\nlazy",
			width: 80,
			want:  []string{"│ quote", "│ lazy"},
		},
		{
			name:  "line ending in emphasis, next starting with strong",
			md:    "A. ends with *emph*\n   **bold** start\n   `code` then `code`\n   last",
			width: 80,
			want: []string{
				"A. ends with emph",
				"   bold start",
				// the style pads a code span with a space on each side
				"    code  then  code",
				"   last",
			},
		},
		{
			name:  "break inside emphasis",
			md:    "A. *emph over\n   two lines* end",
			width: 80,
			want:  []string{"A. emph over", "   two lines end"},
		},
		{
			name:  "no wrap width set",
			md:    letteredOptions,
			width: 0,
			want: []string{
				"The fix:",
				"A. First option, wrapped by hand",
				"   onto an indented line.",
				"B. Second option.",
			},
		},
		{
			name:  "ordered list starting past one",
			md:    "9. nine\n10. ten\n    more",
			width: 80,
			want:  []string{"9. nine", "10. ten", "    more"},
		},
		{
			name:  "checked task",
			md:    "- [x] done\n      more",
			width: 80,
			want:  []string{"[✓] done", "    more"},
		},
		{
			name:  "definition continuation lands under the definition text",
			md:    "Term\n: definition line\n  continued",
			width: 80,
			want:  []string{"Term", "🠶 definition line", "  continued"},
		},
		{
			name:  "list continuation rows keep their indent when they wrap",
			md:    "- item one\n  continued here with many words that run past",
			width: 24,
			want: []string{
				"• item one",
				"  continued here with",
				"  many words that run",
				"  past",
			},
		},
		{
			name:  "a word too long for the indented width falls back to no indent",
			md:    "A. see\n   " + strings.Repeat("x", 30) + " after",
			width: 20,
			want:  []string{"A. see", strings.Repeat("x", 30), "after"},
		},
		{
			name:  "link text is not split by a typed break",
			md:    "See [the\nnotes](notes.md) now.\nNext line.",
			width: 80,
			want:  []string{"See the notes /notes.md now.", "Next line."},
		},
		{
			name:  "hard break keeps the next line's indent",
			md:    "A. one\\\n   two",
			width: 80,
			want:  []string{"A. one", "   two"},
		},
		{
			name:  "hard breaks still break",
			md:    "one\\\ntwo  \nthree",
			width: 80,
			want:  []string{"one", "two", "three"},
		},
		{
			name:  "paragraphs stay separated",
			md:    "one\ntwo\n\nthree",
			width: 80,
			want:  []string{"one", "two", "", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewANSIRendererWithStyle("dark").WithPreservedNewLines().WithWordWrap(tt.width)
			assertLines(t, renderLines(t, r, tt.md), tt.want)
		})
	}
}

func TestRender_PreservedNewLines_LeavesOtherBlocksAlone(t *testing.T) {
	md := "# Heading\n\n" +
		"```go\nfunc main() {\n    println(1)\n}\n```\n\n" +
		"| a | b |\n|---|---|\n| 1 | 2 |\n\n" +
		"Setext\nheading\n=======\n"
	off := NewANSIRendererWithStyle("dark").WithWordWrap(80)
	on := off.WithPreservedNewLines()
	assertLines(t, renderLines(t, on, md), renderLines(t, off, md))
}

func TestRender_PreservedNewLines_ListSecondParagraph(t *testing.T) {
	r := NewANSIRendererWithStyle("dark").WithPreservedNewLines().WithWordWrap(80)
	got := renderLines(t, r, "- first\n\n  second para\n    indented")
	// The renderer glues a list item's later paragraphs onto the first,
	// so only the continuation row is asserted: 2 deeper than its paragraph.
	if !slices.Contains(got, "  indented") {
		t.Errorf("continuation row missing or misaligned: %q", got)
	}
}

func TestANSIStyleRenderer_OptionsKeepPreservedNewLines(t *testing.T) {
	base := NewANSIRendererWithStyle("dark").WithPreservedNewLines()
	derived := map[string]*ANSIStyleRenderer{
		"WithWordWrap":       base.WithWordWrap(40),
		"WithCodeTheme":      base.WithCodeTheme("dracula"),
		"WithCodeBackground": base.WithCodeBackground("#000000"),
		"WithCodeBorder":     base.WithCodeBorder("#111111"),
	}
	for name, r := range derived {
		if !r.preserveNewLines {
			t.Errorf("%s dropped preserveNewLines", name)
		}
	}
	if NewANSIRendererWithStyle("dark").preserveNewLines {
		t.Error("preserveNewLines must default to off")
	}
}

func TestSession_PreserveNewLinesIsDecidedPerPage(t *testing.T) {
	const md = "one\ntwo"
	s := New(Options{})
	s.SetPreserveNewLinesFunc(func(source string) bool { return source == "ticket" })
	s.SetWidth(80)

	lines := func() []string {
		return cleanLines(RenderResult{Lines: s.RenderedLines(), Cleaner: LineCleanerFunc(stripANSIAndMarkers)})
	}

	if err := s.SetMarkdownWithSource(md, "ticket", false); err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines(), []string{"one", "two"})

	if err := s.SetMarkdownWithSource(md, "doc.md", true); err != nil {
		t.Fatal(err)
	}
	assertLines(t, lines(), []string{"one two"})

	s.SetWidth(60)
	assertLines(t, lines(), []string{"one two"})

	if !s.GoBack() {
		t.Fatal("GoBack failed")
	}
	s.SetWidth(70)
	assertLines(t, lines(), []string{"one", "two"})
}

func TestSession_NoPreserveNewLinesFuncJoins(t *testing.T) {
	s := New(Options{})
	s.SetWidth(80)
	if err := s.SetMarkdownWithSource("one\ntwo", "ticket", false); err != nil {
		t.Fatal(err)
	}
	got := cleanLines(RenderResult{Lines: s.RenderedLines(), Cleaner: LineCleanerFunc(stripANSIAndMarkers)})
	assertLines(t, got, []string{"one two"})
}

func TestSession_LinkAcrossATypedBreakStaysSelectable(t *testing.T) {
	s := New(Options{})
	s.SetPreserveNewLinesFunc(func(string) bool { return true })
	s.SetWidth(80)
	md := "See [the\nnotes](notes.md) now.\nThen [second](two.md) here."
	if err := s.SetMarkdownWithSource(md, "ticket", false); err != nil {
		t.Fatal(err)
	}

	var got []string
	lines := map[string]int{}
	for s.MoveToNextLink(20) {
		sel := s.Selected()
		if sel == nil || lines[sel.URL] != 0 || len(got) > 4 {
			break
		}
		got = append(got, sel.URL)
		lines[sel.URL] = sel.StartLine + 1
	}
	if want := []string{"notes.md", "two.md"}; !slices.Equal(got, want) {
		t.Fatalf("links reached = %v, want %v", got, want)
	}
	if lines["notes.md"] == lines["two.md"] {
		t.Errorf("both links placed on rendered line %d", lines["two.md"]-1)
	}
}
