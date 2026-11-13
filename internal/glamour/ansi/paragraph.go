package ansi

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/muesli/reflow/wordwrap"
)

// A ParagraphElement is used to render individual paragraphs.
type ParagraphElement struct {
	First bool
}

// Render renders a ParagraphElement.
func (e *ParagraphElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := ctx.options.Styles.Paragraph

	if !e.First {
		_, _ = io.WriteString(w, "\n")
	}
	be := BlockElement{
		Block: &bytes.Buffer{},
		Style: cascadeStyle(bs.Current().Style, rules, false),
	}
	bs.Push(be)

	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockPrefix)
	renderText(bs.Current().Block, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Prefix)
	return nil
}

// Finish finishes rendering a ParagraphElement.
func (e *ParagraphElement) Finish(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack
	rules := bs.Current().Style

	mw := NewMarginWriter(ctx, w, rules)
	if len(strings.TrimSpace(bs.Current().Block.String())) > 0 {
		wrapped, err := e.wrap(bs.Current().Block.Bytes(), int(bs.Width(ctx)), ctx) //nolint: gosec
		if err != nil {
			return err
		}

		_, err = mw.Write(wrapped)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(mw, "\n")
	}

	renderText(w, ctx.options.ColorProfile, bs.Current().Style.StylePrimitive, rules.Suffix)
	renderText(w, ctx.options.ColorProfile, bs.Parent().Style.StylePrimitive, rules.BlockSuffix)

	bs.Current().Block.Reset()
	bs.Pop()
	return nil
}

func (e *ParagraphElement) wrap(text []byte, width int, ctx RenderContext) ([]byte, error) {
	if ctx.options.PreserveNewLines && width > 0 {
		wrapped, err := wrapKeepingIndent(string(text), width, wrapWords)
		return []byte(wrapped), err
	}
	flow := wordwrap.NewWriter(width)
	flow.KeepNewlines = true
	_, _ = flow.Write(text)
	if err := flow.Close(); err != nil {
		return nil, fmt.Errorf("glamour: error closing flow: %w", err)
	}
	return flow.Bytes(), nil
}
