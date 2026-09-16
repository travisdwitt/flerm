package tui

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Golden-file rendering harness. Regenerate with:
//
//	go test ./internal/tui -run TestSnapshots -update
//
// The .txt files under testdata/ are the app's actual screen output with ANSI
// stripped, so a visual change to any view shows up as a reviewable diff.

var updateSnapshots = flag.Bool("update", false, "rewrite testdata snapshots")

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;?]*[a-zA-Z]")

func snapshot(t *testing.T, name string, m model) {
	t.Helper()
	got := ansiRE.ReplaceAllString(m.View(), "") + "\n"
	path := filepath.Join("testdata", name+".txt")
	if *updateSnapshots {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v (run with -update to create)", name, err)
	}
	if string(want) != got {
		t.Errorf("%s changed; rerun with -update to accept:\n--- got ---\n%s", name, got)
	}
}

// chartModel builds a representative flowchart: styled boxes, a titled box,
// arrowed connections, a free text label, and a highlight.
func chartModel(w, h int) model {
	m := initialModel()
	m.mode = ModeNormal
	m.width, m.height = w, h
	c := m.getCanvas()
	c.AddBox(4, 2, "Start")
	c.AddBox(4, 10, "Process\nthe input")
	c.AddBox(34, 10, "Decision?")
	c.AddBox(34, 2, "Done")
	c.Boxes()[1].Title = "Step 1"
	c.Boxes()[1].UpdateSize()
	c.SetBorderStyle(2, BorderStyleRounded)
	c.SetBorderStyle(3, BorderStyleDouble)
	c.AddConnection(0, 1)
	c.AddConnection(1, 2)
	c.AddConnection(2, 3)
	for i := range c.Connections() {
		c.Connections()[i].ArrowTo = true
	}
	c.AddText(4, 18, "note: label text")
	c.SetHighlight(5, 18, 1)
	return m
}

func TestSnapshots(t *testing.T) {
	snapshot(t, "chart_wide", chartModel(96, 30))
	snapshot(t, "chart_narrow", chartModel(46, 16))
	snapshot(t, "chart_tiny", chartModel(20, 6))

	sel := chartModel(96, 30)
	sel.selectedBox = 1
	sel.selBox = 1
	sel.cursorX, sel.cursorY = 6, 11
	snapshot(t, "chart_selected", sel)

	help := chartModel(96, 30)
	help.help = true
	snapshot(t, "help", help)

	start := initialModel()
	start.mode = ModeStartup
	start.width, start.height = 96, 30
	start.lastFile = "" // whatever this machine last opened must not leak in
	snapshot(t, "startup", start)

	resume := start
	resume.lastFile = "/tmp/quarterly-plan.sav"
	snapshot(t, "startup_resume", resume)

	save := chartModel(96, 30)
	save.mode = ModeFileInput
	save.fileOp = FileOpSave
	save.filename = "quarterly-plan"
	snapshot(t, "dialog_save", save)

	// Height 18 leaves a 10-row list window, so the list still scrolls.
	openDlg := chartModel(96, 18)
	openDlg.mode = ModeFileInput
	openDlg.fileOp = FileOpOpen
	for _, n := range []string{"alpha", "beta", "delta", "gamma", "omega", "sigma",
		"theta", "zeta", "kappa", "lambda", "mu", "nu"} {
		openDlg.allFiles = append(openDlg.allFiles, n+".sav")
	}
	openDlg.selectedFileIndex = 11
	openDlg.applyFileFilter()
	snapshot(t, "dialog_open_scrolled", openDlg)

	search := openDlg
	search.fileSearch = true
	search.fileFilter = "ma"
	search.refilterFromTop()
	snapshot(t, "dialog_open_search", search)

	confirm := chartModel(96, 30)
	confirm.mode = ModeConfirm
	confirm.confirmAction = ConfirmQuit
	snapshot(t, "dialog_confirm", confirm)

	errm := chartModel(96, 30)
	errm.errorMessage = "could not write /nonexistent/dir/chart.sav: permission denied"
	snapshot(t, "message_error", errm)
}

// TestViewNeverPanics is the cheap guard: every mode must render at every
// plausible terminal size, including degenerate ones.
func TestViewNeverPanics(t *testing.T) {
	sizes := [][2]int{{0, 0}, {1, 1}, {8, 3}, {20, 6}, {40, 10}, {80, 24}, {300, 100}}
	modes := []Mode{ModeStartup, ModeNormal, ModeEditing, ModeTextInput, ModeResize,
		ModeMove, ModeMultiSelect, ModeFileInput, ModeConfirm, ModeBoxJump,
		ModeTitleEdit, ModeContextMenu}
	for _, sz := range sizes {
		for _, mode := range modes {
			for _, help := range []bool{false, true} {
				m := chartModel(sz[0], sz[1])
				m.mode = mode
				m.help = help
				m.selectedBox, m.selectedText = 1, -1
				m.menuItems = []MenuItem{{Label: "New Box"}, {Separator: true}, {Label: "Delete"}}
				m.errorMessage = "some error"
				_ = m.View()
			}
		}
	}
}
