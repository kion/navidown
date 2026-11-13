package ansi

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/muesli/reflow/wordwrap"
	"github.com/yuin/goldmark/ast"
	astext "github.com/yuin/goldmark/extension/ast"
)

const tabStop = 4

// lineBlock returns the paragraph or text block whose source lines hold n.
// Other blocks (headings, table cells) keep joining their lines.
func lineBlock(n ast.Node) ast.Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case ast.KindParagraph, ast.KindTextBlock:
			return p
		case ast.KindLink, ast.KindImage:
			// link text split over two rows can no longer be located and selected
			return nil
		}
		if p.Type() != ast.TypeInline {
			return nil
		}
	}
	return nil
}

// continuationIndent returns how far the line after n is indented in the source,
// relative to where the block's first line is drawn.
// The parser strips that indent from the text,
// so it is recovered from the block's line offsets.
func continuationIndent(n *ast.Text, block ast.Node, source []byte, styles StyleConfig) int {
	lines := block.Lines()
	if lines == nil || lines.Len() < 2 {
		return 0
	}
	for i := 1; i < lines.Len(); i++ {
		next := lines.At(i)
		if next.Start <= n.Segment.Stop {
			continue
		}
		drawnAt := sourceColumn(source, lines.At(0).Start) - markerWidth(block, styles)
		return max(0, sourceColumn(source, next.Start)-drawnAt)
	}
	return 0
}

func sourceColumn(source []byte, offset int) int {
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	col := 0
	for _, b := range source[start:offset] {
		if b == '\t' {
			col += tabStop - col%tabStop
		} else {
			col++
		}
	}
	return col
}

// markerWidth is the width of what a list item or definition draws before its first block,
// measured against what the source has there.
func markerWidth(block ast.Node, styles StyleConfig) int {
	if block.Parent().Kind() == astext.KindDefinitionDescription && block.Parent().FirstChild() == block {
		return primitiveLeadWidth(styles.DefinitionDescription)
	}
	item, ok := block.Parent().(*ast.ListItem)
	if !ok || item.FirstChild() != block {
		return 0
	}
	list, ok := item.Parent().(*ast.List)
	if !ok {
		return 0
	}
	if box, ok := block.FirstChild().(*astext.TaskCheckBox); ok {
		tick := styles.Task.Unticked
		if box.IsChecked {
			tick = styles.Task.Ticked
		}
		// the source checkbox "[ ] " sits inside the block's first line
		return xansi.StringWidth(tick) - len("[ ] ")
	}
	if list.IsOrdered() {
		number := strconv.FormatUint(uint64(listItemNumber(item, list)), 10)
		return len(number) + primitiveLeadWidth(styles.Enumeration)
	}
	return primitiveLeadWidth(styles.Item)
}

func primitiveLeadWidth(p StylePrimitive) int {
	return xansi.StringWidth(p.BlockPrefix) + xansi.StringWidth(p.Prefix)
}

func listItemNumber(item ast.Node, list *ast.List) uint {
	var n uint = 1
	for s := item.PreviousSibling(); s != nil && s.Kind() == ast.KindListItem; s = s.PreviousSibling() {
		n++
	}
	if list.Start != 1 {
		n += uint(list.Start) - 1 //nolint: gosec
	}
	return n
}

type wrapFunc func(line string, width int) (string, error)

func wrapWords(line string, width int) (string, error) {
	flow := wordwrap.NewWriter(width)
	_, _ = flow.Write([]byte(line))
	if err := flow.Close(); err != nil {
		return "", fmt.Errorf("glamour: error closing flow: %w", err)
	}
	return flow.String(), nil
}

// wrapKeepingIndent wraps each line on its own,
// so the rows a line wraps into keep that line's leading indent.
func wrapKeepingIndent(text string, width int, wrap wrapFunc) (string, error) {
	var out strings.Builder
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			out.WriteByte('\n')
		}
		rest := strings.TrimLeft(line, " ")
		indent := len(line) - len(rest)

		rows, err := wrapIndented(rest, width, indent, wrap)
		if err != nil {
			return "", err
		}
		out.WriteString(rows)
	}
	return out.String(), nil
}

func wrapIndented(line string, width, indent int, wrap wrapFunc) (string, error) {
	if indent > 0 && indent < width {
		wrapped, err := wrap(line, width-indent)
		if err != nil {
			return "", err
		}
		rows := strings.Split(wrapped, "\n")
		if rowsFit(rows, width-indent) {
			pad := strings.Repeat(" ", indent)
			return pad + strings.Join(rows, "\n"+pad), nil
		}
		// a word too long for the narrower width would be re-broken by the enclosing block,
		// stranding the indent on a row of its own
	}
	return wrap(line, width)
}

func rowsFit(rows []string, width int) bool {
	for _, row := range rows {
		if xansi.StringWidth(row) > width {
			return false
		}
	}
	return true
}
