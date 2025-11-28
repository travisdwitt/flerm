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
	buf := m.getCurrentBuffer()
	if buf == nil {
		return nil
	}
	return buf.canvas
}

func (m *model) getPanOffset() (int, int) {
	buf := m.getCurrentBuffer()
	if buf == nil {
		return 0, 0
	}
	return buf.panX, buf.panY
}

func (m *model) worldCoords() (int, int) {
	panX, panY := m.getPanOffset()
	return m.cursorX + panX, m.cursorY + panY
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

// readClipboardText reads plain text from the clipboard
// Returns text exactly as it exists in the clipboard with NO processing
func readClipboardText() (string, error) {
	// On macOS, use pbpaste -Prefer txt to get plain text
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbpaste", "-Prefer", "txt")
		output, err := cmd.Output()
		if err == nil {
			// Return exactly as-is, no processing whatsoever
			return string(output), nil
		}
		// If that fails, try regular pbpaste
		cmd = exec.Command("pbpaste")
		output, err = cmd.Output()
		if err == nil {
			// Return exactly as-is, no processing whatsoever
			return string(output), nil
		}
	}
	
	// On all platforms, use clipboard library as fallback
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	// Return exactly as-is, no processing whatsoever
	return text, nil
}

// isRTF checks if the text appears to be RTF formatted
func isRTF(text string) bool {
	return strings.HasPrefix(text, "{\\rtf") || strings.Contains(text, "\\rtf1")
}

// isHTML checks if the text appears to be HTML formatted
func isHTML(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "<") && 
		   (strings.Contains(text, "<html") || strings.Contains(text, "<body") || strings.Contains(text, "<div"))
}

// extractTextFromRTF extracts plain text from RTF formatted text
// This is a more aggressive parser that only keeps actual text content
func extractTextFromRTF(rtf string) string {
	var result strings.Builder
	result.Grow(len(rtf))
	
	bytes := []byte(rtf)
	inControl := false
	
	for i := 0; i < len(bytes); i++ {
		b := bytes[i]
		
		// Skip RTF group delimiters completely
		if b == '{' || b == '}' {
			inControl = false
			continue
		}
		
		// Handle backslash - start of RTF control sequence
		if b == '\\' {
			inControl = true
			if i+1 < len(bytes) {
				next := bytes[i+1]
				
				// Check for hex character code: \'hh
				if next == '\'' && i+3 < len(bytes) {
					hexStr := string(bytes[i+2 : i+4])
					if val, err := strconv.ParseUint(hexStr, 16, 8); err == nil {
						result.WriteByte(byte(val))
						i += 3 // Skip \'hh
						inControl = false
						continue
					}
				}
				
				// Check for escaped characters: \\ \{ \}
				if next == '\\' {
					result.WriteByte('\\')
					i++ // Skip both
					inControl = false
					continue
				}
				if next == '{' {
					result.WriteByte('{')
					i++ // Skip both
					inControl = false
					continue
				}
				if next == '}' {
					result.WriteByte('}')
					i++ // Skip both
					inControl = false
					continue
				}
				
				// Check for special symbols
				if next == '-' {
					result.WriteByte('-') // Non-breaking hyphen
					i++ // Skip both
					inControl = false
					continue
				}
				if next == '_' {
					result.WriteByte(' ') // Non-breaking space
					i++ // Skip both
					inControl = false
					continue
				}
				
				// Check for control words
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
					// Build control word
					start := i + 1
					for i+1 < len(bytes) {
						c := bytes[i+1]
						if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
							i++
							continue
						}
						// Check for numeric parameter
						if c == '-' || (c >= '0' && c <= '9') {
							for i+1 < len(bytes) {
								nc := bytes[i+1]
								if nc == '-' || (nc >= '0' && nc <= '9') {
									i++
									continue
								}
								break
							}
						}
						// Skip optional space
						if i+1 < len(bytes) && bytes[i+1] == ' ' {
							i++
						}
						break
					}
					
					// Check if this is a text control word we care about
					if i >= start {
						word := string(bytes[start : i+1])
						if word == "par" || word == "line" {
							result.WriteByte('\n')
						} else if word == "tab" {
							result.WriteByte('\t')
						}
						// All other control words are skipped
					}
					inControl = false
					continue
				}
			}
			// Unknown backslash sequence - skip it
			inControl = false
			continue
		}
		
		// If we're in a control sequence, skip this byte
		if inControl {
			continue
		}
		
		// Regular character - keep it if it's printable
		if b >= 32 && b < 127 {
			result.WriteByte(b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			result.WriteByte(b)
		}
		// Skip other control characters
	}
	
	return result.String()
}

// extractTextFromHTML extracts plain text from HTML (basic implementation)
func extractTextFromHTML(html string) string {
	// Simple HTML tag removal - keep text content
	var result strings.Builder
	result.Grow(len(html))
	
	inTag := false
	runes := []rune(html)
	
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		
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
	
	// Decode HTML entities
	text := result.String()
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	
	return text
}

// cleanClipboardText removes control characters and formatting from clipboard text
// while preserving actual text content, newlines, and tabs
func cleanClipboardText(text string) string {
	if text == "" {
		return text
	}
	
	// First, try to detect and strip RTF formatting if present
	text = stripRTF(text)
	
	var result strings.Builder
	result.Grow(len(text))
	
	for _, r := range text {
		// Allow newlines, carriage returns, and tabs
		if r == '\n' || r == '\r' || r == '\t' {
			result.WriteRune(r)
		} else if r >= 32 {
			// Allow all printable characters (ASCII and Unicode)
			// This includes all normal text characters, symbols, and Unicode text
			result.WriteRune(r)
		}
		// Skip only control characters (0-31 except \n, \r, \t)
		// This removes formatting markers while keeping actual text
	}
	
	// Normalize line endings: convert \r\n to \n, and standalone \r to \n
	normalized := result.String()
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	
	return normalized
}

// stripRTF attempts to extract plain text from RTF formatted text
func stripRTF(text string) string {
	// Check if this looks like RTF
	if !strings.HasPrefix(text, "{\\rtf") && !strings.Contains(text, "\\rtf") {
		return text
	}
	
	// Simple RTF stripping: extract text content while skipping RTF control codes
	var result strings.Builder
	result.Grow(len(text))
	
	runes := []rune(text)
	inControl := false
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
			// RTF control sequence
			if i+1 < len(runes) {
				next := runes[i+1]
				// Check if it's a control word (letters) or control symbol (single char)
				if (next >= 'a' && next <= 'z') || (next >= 'A' && next <= 'Z') {
					// Control word - skip until space or non-letter
					i++ // Skip the backslash
					for i < len(runes) {
						if runes[i] == ' ' || runes[i] == '\\' || runes[i] == '{' || runes[i] == '}' {
							if runes[i] == ' ' {
								i++ // Skip the space
							}
							break
						}
						i++
					}
					i-- // Adjust for loop increment
					continue
				} else if next == '\\' || next == '{' || next == '}' {
					// Escaped character - keep it
					result.WriteRune(next)
					i++ // Skip both backslash and escaped char
					continue
				} else if next == '\n' || next == '\r' || next == '\t' {
					// Control symbol for newline/carriage return/tab
					result.WriteRune(next)
					i++ // Skip both backslash and control char
					continue
				}
			}
			// Unknown control - skip the backslash
			continue
		}
		
		// Regular character - keep it if we're not in a control sequence
		if !inControl {
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

