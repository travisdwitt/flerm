package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
)

func (m *model) getCurrentBuffer() *Buffer {
	if len(m.buffers) == 0 {
		return nil
	}
	return &m.buffers[m.currentBufferIndex]
}

func (m *model) getCanvas() *Canvas {
	if buf := m.getCurrentBuffer(); buf != nil {
		return buf.canvas
	}
	return nil
}

func (m *model) getPanOffset() (int, int) {
	if buf := m.getCurrentBuffer(); buf != nil {
		return buf.panX, buf.panY
	}
	return 0, 0
}

func (m *model) worldCoords() (int, int) {
	panX, panY := m.getPanOffset()
	return m.cursorX + panX, m.cursorY + panY
}

func (m *model) getWorldCoordsAt(cursorX, cursorY int) (int, int) {
	panX, panY := m.getPanOffset()
	return cursorX + panX, cursorY + panY
}

func (m *model) addNewBuffer(canvas *Canvas, filename string) {
	m.addNewBufferWithPan(canvas, filename, 0, 0)
}

func (m *model) addNewBufferWithPan(canvas *Canvas, filename string, panX, panY int) {
	buffer := Buffer{
		canvas:    canvas,
		undoStack: []Action{},
		redoStack: []Action{},
		filename:  filename,
		panX:      panX,
		panY:      panY,
	}
	m.buffers = append(m.buffers, buffer)
	m.currentBufferIndex = len(m.buffers) - 1
}

func (m *model) recordAction(actionType ActionType, data, inverse interface{}) {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return
	}
	action := Action{
		Type:    actionType,
		Data:    data,
		Inverse: inverse,
	}
	buf.undoStack = append(buf.undoStack, action)
	buf.redoStack = buf.redoStack[:0]
}

func readClipboardText() (string, error) {
	if runtime.GOOS == "darwin" {
		if output, err := exec.Command("pbpaste", "-Prefer", "txt").Output(); err == nil {
			return string(output), nil
		}
		if output, err := exec.Command("pbpaste").Output(); err == nil {
			return string(output), nil
		}
	}
	return clipboard.ReadAll()
}

func isRTF(text string) bool {
	return strings.HasPrefix(text, "{\\rtf") || strings.Contains(text, "\\rtf1")
}

func isHTML(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "<") &&
		(strings.Contains(text, "<html") || strings.Contains(text, "<body") || strings.Contains(text, "<div"))
}

func extractTextFromRTF(rtf string) string {
	var result strings.Builder
	result.Grow(len(rtf))
	bytes := []byte(rtf)
	inControl := false

	for i := 0; i < len(bytes); i++ {
		b := bytes[i]
		if b == '{' || b == '}' {
			inControl = false
			continue
		}
		if b == '\\' {
			inControl = true
			if i+1 < len(bytes) {
				next := bytes[i+1]
				if next == '\'' && i+3 < len(bytes) {
					if val, err := strconv.ParseUint(string(bytes[i+2:i+4]), 16, 8); err == nil {
						result.WriteByte(byte(val))
						i += 3
						inControl = false
						continue
					}
				}
				if next == '\\' || next == '{' || next == '}' {
					result.WriteByte(next)
					i++
					inControl = false
					continue
				}
				if next == '-' {
					result.WriteByte('-')
					i++
					inControl = false
					continue
				}
				if next == '_' {
					result.WriteByte(' ')
					i++
					inControl = false
					continue
				}
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
					start := i + 1
					for i+1 < len(bytes) {
						c := bytes[i+1]
						if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
							i++
							continue
						}
						if c == '-' || (c >= '0' && c <= '9') {
							for i+1 < len(bytes) {
								if nc := bytes[i+1]; nc == '-' || (nc >= '0' && nc <= '9') {
									i++
									continue
								}
								break
							}
						}
						if i+1 < len(bytes) && bytes[i+1] == ' ' {
							i++
						}
						break
					}
					if i >= start {
						word := string(bytes[start : i+1])
						if word == "par" || word == "line" {
							result.WriteByte('\n')
						} else if word == "tab" {
							result.WriteByte('\t')
						}
					}
					inControl = false
					continue
				}
			}
			inControl = false
			continue
		}
		if inControl {
			continue
		}
		if b >= 32 && b < 127 {
			result.WriteByte(b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			result.WriteByte(b)
		}
	}
	return result.String()
}

func extractTextFromHTML(html string) string {
	var result strings.Builder
	result.Grow(len(html))
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	return text
}

func cleanClipboardText(text string) string {
	if text == "" {
		return text
	}
	text = stripRTF(text)
	var result strings.Builder
	result.Grow(len(text))
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' || r >= 32 {
			result.WriteRune(r)
		}
	}
	normalized := result.String()
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return normalized
}

func stripRTF(text string) string {
	if !strings.HasPrefix(text, "{\\rtf") && !strings.Contains(text, "\\rtf") {
		return text
	}
	var result strings.Builder
	result.Grow(len(text))
	runes := []rune(text)
	braceDepth := 0
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '{' {
			braceDepth++
			continue
		}
		if r == '}' {
			braceDepth--
			continue
		}
		if r == '\\' {
			if i+1 < len(runes) {
				next := runes[i+1]
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
					i++
					for i < len(runes) {
						if runes[i] == ' ' || runes[i] == '\\' || runes[i] == '{' || runes[i] == '}' {
							if runes[i] == ' ' {
								i++
							}
							break
						}
						i++
					}
					i--
					continue
				} else if next == '\\' || next == '{' || next == '}' {
					result.WriteRune(next)
					i++
					continue
				} else if next == '\n' || next == '\r' || next == '\t' {
					result.WriteRune(next)
					i++
					continue
				}
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

