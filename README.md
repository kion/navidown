# navidown

## About

`navidown` is a reusable navigable markdown CLI component:

- renders markdown to **ANSI** (for terminal output)
- supports **scrolling** and **pager-style navigation**
- finds **links** and allows **Tab / Shift-Tab** traversal
- on activation (Enter), loads linked markdown via a pluggable loader and replaces current content

This repo contains:

- core package `github.com/boolean-maybe/navidown/navidown` (UI-agnostic state machine + renderers)
- optional TView adapter `github.com/boolean-maybe/navidown/navidown/tview` (TView primitive)

## Installation

```bash
go get github.com/boolean-maybe/navidown/navidown
```

For the TView adapter:
```bash
go get github.com/boolean-maybe/navidown/navidown/tview
```

For the file/HTTP content loader:
```bash
go get github.com/boolean-maybe/navidown/loaders
```

## Quick Start

### Using the TView Adapter

Two adapters are available: `BoxViewer` (custom draw) and `TextViewViewer` (TextView paging). This example uses `TextViewViewer`.

```go
package main

import (
	"github.com/boolean-maybe/navidown/loaders"
	nav "github.com/boolean-maybe/navidown/navidown"
	navtview "github.com/boolean-maybe/navidown/navidown/tview"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()
	viewer := navtview.NewTextView()

	// Set initial markdown content
	viewer.SetMarkdown(`# Welcome to navidown

Navigate through links with **Tab** and **Shift+Tab**.
Press **Enter** to follow links.

[Example Link](https://example.com)
[Local File](./README.md)

## Features
- ANSI terminal rendering
- Link navigation
- Scrolling support
`)

	// Handle link activation
	fetcher := nav.NewContentFetcher(&loaders.FileHTTP{}, nil)
	viewer.SetSelectHandler(func(v *navtview.TextViewViewer, elem nav.NavElement) {
		content, err := fetcher.FetchContent(elem)
		if err == nil {
			v.SetMarkdownWithSource(content, elem.URL, true)
		}
	})

	if err := app.SetRoot(viewer, true).Run(); err != nil {
		panic(err)
	}
}
```

### Using the Core API (UI-agnostic)

```go
package main

import (
	"fmt"
	"github.com/boolean-maybe/navidown/navidown"
)

func main() {
	session := navidown.New(navidown.Options{})

	// Set markdown content
	_ = session.SetMarkdown(`# Hello World

This is a [link](https://example.com).

## Section
More content here.`)

	// Get rendered ANSI lines
	lines := session.RenderedLines()
	for _, line := range lines {
		fmt.Println(line)
	}

	// Navigate through links
	session.MoveToNextLink(20)
	if selected := session.Selected(); selected != nil {
		fmt.Printf("Selected: %s -> %s\n", selected.Text, selected.URL)
	}
}
```

### Keeping the author's line breaks

By default a paragraph's line breaks are joined and the text is re-wrapped, as Markdown renderers do.
A renderer can instead keep them, along with the indentation of continuation lines:

```go
renderer := navidown.NewANSIRendererWithStyle("dark").WithPreservedNewLines()
```

With that option a block such as

```markdown
The options:
A. First option, wrapped by hand
   onto an indented line.
B. Second option.
```

renders as four rows. A line longer than the width still wraps, and the wrapped rows keep that line's
indent. Headings, tables and code blocks are unaffected.

A session can decide page by page, which suits a viewer that shows both hand-laid-out notes and
ordinary documents. The function receives the source path of each page as it is rendered:

```go
session.SetPreserveNewLinesFunc(func(sourceFilePath string) bool {
	return strings.HasSuffix(sourceFilePath, ".note.md")
})
```

![Build Status](https://github.com/boolean-maybe/navidown/actions/workflows/go.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/boolean-maybe/navidown)](https://goreportcard.com/report/github.com/boolean-maybe/navidown)
[![Go Reference](https://pkg.go.dev/badge/github.com/boolean-maybe/navidown.svg)](https://pkg.go.dev/github.com/boolean-maybe/navidown)