package canvas

import (
	"os"
	"strings"
)

type ColorMode int

const (
	ColorModeNone ColorMode = iota
	ColorMode16
	ColorMode256
)

var colorMode = detectColorMode()

func detectColorMode() ColorMode {
	term := os.Getenv("TERM")
	switch {
	case os.Getenv("NO_COLOR") != "", term == "", term == "dumb":
		return ColorModeNone
	case strings.Contains(term, "256color"), os.Getenv("COLORTERM") != "":
		return ColorMode256
	}
	return ColorMode16
}

func ColorEnabled() bool { return colorMode != ColorModeNone }

func Ansi(code string) string {
	if colorMode == ColorModeNone {
		return ""
	}
	return code
}

func AnsiTier(code256, code16 string) string {
	switch colorMode {
	case ColorModeNone:
		return ""
	case ColorMode256:
		return code256
	}
	return code16
}

func ColorFG(index int) string { return colorCode(index, false) }
