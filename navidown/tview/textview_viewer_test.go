package tview

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestViewerReportsWhetherItHasBeenDrawn(t *testing.T) {
	v := NewTextView()
	if v.Drawn() {
		t.Fatal("a new viewer reports it has been drawn")
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	v.SetRect(0, 0, 60, 20)
	v.Draw(screen)

	if !v.Drawn() {
		t.Fatal("a drawn viewer reports it has not been drawn")
	}
}
