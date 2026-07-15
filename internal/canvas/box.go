package canvas

import (
	"strings"
)

type Text struct {
	X     int
	Y     int
	Lines []string
	ID    int
	Color int
}

func (t *Text) GetText() string {
	return strings.Join(t.Lines, "\n")
}

func (t *Text) SetText(text string) {
	t.Lines = strings.Split(text, "\n")
}

type Box struct {
	X            int
	Y            int
	Width        int
	Height       int
	Lines        []string
	ID           int
	ZLevel       int
	BorderStyle  BorderStyle
	OriginalText string
	Title        string
	Color        int
}

func (b *Box) GetText() string {
	if b.OriginalText != "" {
		return b.OriginalText
	}
	return strings.Join(b.Lines, "\n")
}

func (b *Box) SetText(text string) {
	b.OriginalText = text
	b.Lines = strings.Split(text, "\n")
	b.UpdateSize()
}

func (b *Box) UpdateSize() {
	if len(b.Lines) == 0 {
		b.Lines = []string{""}
	}

	maxWidth := minBoxWidth

	titleLines := []string{}
	if b.Title != "" {
		titleLines = strings.Split(b.Title, "\n")
		for _, titleLine := range titleLines {
			titleWidth := len(titleLine) + 2
			if titleWidth > maxWidth {
				maxWidth = titleWidth
			}
		}
	}

	for _, line := range b.Lines {
		if len(line)+2 > maxWidth {
			maxWidth = len(line) + 2
		}
	}
	b.Width = maxWidth

	extraHeight := 0
	if b.Title != "" {
		extraHeight = len(titleLines) + 1
	}
	b.Height = len(b.Lines) + 2 + extraHeight
}

func (b *Box) fitTextToSize(newWidth, newHeight int) {
	text := b.GetText()
	if text == "" {
		b.Lines = []string{""}
		return
	}

	contentWidth := newWidth - 2
	contentHeight := newHeight - 2

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	originalLines := strings.Split(text, "\n")

	fitsWidth := true
	for _, line := range originalLines {
		if len(line) > contentWidth {
			fitsWidth = false
			break
		}
	}

	fitsHeight := len(originalLines) <= contentHeight

	if fitsWidth && fitsHeight {
		b.Lines = originalLines
		return
	}

	var resultLines []string

	for i, line := range originalLines {
		if i >= contentHeight {

			break
		}

		if i == contentHeight-1 && (len(originalLines) > contentHeight || !fitsWidth) {

			if len(line) > contentWidth-3 {
				line = line[:contentWidth-3] + "..."
			} else if len(originalLines) > contentHeight {

				if len(line)+3 <= contentWidth {
					line = line + "..."
				} else {
					line = line[:contentWidth-3] + "..."
				}
			}
		} else if len(line) > contentWidth {

			if contentWidth > 3 {
				line = line[:contentWidth-3] + "..."
			} else {
				line = line[:contentWidth]
			}
		}

		resultLines = append(resultLines, line)
	}

	if len(resultLines) == 0 {
		resultLines = []string{""}
	}

	b.Lines = resultLines
}

func (b *Box) IsTextTruncated() bool {
	if b.OriginalText == "" {
		return false
	}

	originalLines := strings.Split(b.OriginalText, "\n")

	if len(originalLines) != len(b.Lines) {
		return true
	}

	for _, line := range b.Lines {
		if strings.HasSuffix(line, "...") {
			return true
		}
	}

	return false
}
