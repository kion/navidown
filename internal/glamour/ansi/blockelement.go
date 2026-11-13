package ansi

import (
	"bytes"
	"fmt"
	"io"

	"github.com/charmbracelet/x/ansi"
)

// BlockElement provides a render buffer for children of a block element.
// After all children have been rendered into it, it applies indentation and
// margins around them and writes everything to the parent rendering buffer.
type BlockElement struct {
	Block   *bytes.Buffer
	Style   StyleBlock
	Margin  bool
	Newline bool
	// KeepIndent wraps line by line, so a wrapped row keeps its line's leading indent.
	KeepIndent bool
}

// Render renders a BlockElement.
func (e *BlockElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	bs.Push(*e)

	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, e.Style.BlockPrefix)
	renderText(bs.Current().Block, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, e.Style.Prefix)
	return nil
}

// Finish finishes rendering a BlockElement.
func (e *BlockElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	if e.Margin { //nolint: nestif
		width := int(bs.Width(ctx)) //nolint: gosec
		s := bs.Current().Block.String()
		if e.KeepIndent && width > 0 {
			var err error
			if s, err = wrapKeepingIndent(s, width, wrapBlock); err != nil {
				return err
			}
		} else {
			s = wrapBlockText(s, width)
		}

		mw := NewMarginWriter(ctx, w, bs.Current().Style)
		if _, err := io.WriteString(mw, s); err != nil {
			return fmt.Errorf("glamour: error writing to writer: %w", err)
		}

		if e.Newline {
			if _, err := io.WriteString(mw, "\n"); err != nil {
				return fmt.Errorf("glamour: error writing to writer: %w", err)
			}
		}
	} else {
		_, err := bs.Parent().Block.Write(bs.Current().Block.Bytes())
		if err != nil {
			return fmt.Errorf("glamour: error writing to writer: %w", err)
		}
	}

	renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, e.Style.Suffix)
	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, e.Style.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}

const blockBreakpoints = " ,.;-+|"

func wrapBlockText(s string, width int) string {
	return ansi.Wordwrap(s, width, blockBreakpoints)
}

func wrapBlock(line string, width int) (string, error) {
	return wrapBlockText(line, width), nil
}
