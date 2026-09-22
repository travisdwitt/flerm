package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiPattern.ReplaceAllString(s, "") }

func startupModel(effect effectKind) model {
	m := initialModel()
	m.mode = ModeStartup
	m.width = 100
	m.height = 34
	m.lastFile = ""
	m.effect = effect
	return m
}

func runFrames(m model, n int) model {
	for i := 0; i < n; i++ {
		m.advanceEffect()
	}
	return m
}

func TestEffectForDate(t *testing.T) {
	tests := []struct {
		name  string
		month time.Month
		day   int
		want  effectKind
	}{
		{"christmas", time.December, 25, effectSnow},
		{"new year", time.January, 1, effectFireworks},
		{"independence day", time.July, 4, effectFireworksUSA},
		{"july 3", time.July, 3, effectNone},
		{"halloween", time.October, 31, effectBlood},
		{"september 17", time.September, 17, effectConfetti},
		{"boxing day", time.December, 26, effectNone},
		{"christmas eve", time.December, 24, effectNone},
		{"ordinary day", time.March, 8, effectNone},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectForDate(time.Date(2026, tc.month, tc.day, 12, 0, 0, 0, time.UTC)); got != tc.want {
				t.Fatalf("effectForDate = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEffectForOverride(t *testing.T) {
	tests := []struct {
		arg  string
		want effectKind
	}{
		{"1225", effectSnow},
		{"0101", effectFireworks},
		{"0704", effectFireworksUSA},
		{"1031", effectBlood},
		{"0917", effectConfetti},
		{"0308", effectNone},
		{"", effectNone},
		{"122", effectNone},
		{"1325", effectNone},
		{"1200", effectNone},
		{"ab12", effectNone},
	}
	for _, tc := range tests {
		t.Run(tc.arg, func(t *testing.T) {
			if got := effectForOverride(tc.arg); got != tc.want {
				t.Fatalf("effectForOverride(%q) = %v, want %v", tc.arg, got, tc.want)
			}
		})
	}
}

func TestChristmasLogoOnlyOnChristmas(t *testing.T) {
	xmas := startupModel(effectSnow).startupLayout()
	if len(xmas.logo) != len(christmasLogo) || xmas.logo[0] != christmasLogo[0] {
		t.Fatalf("expected the christmas logo, got %q", xmas.logo[0])
	}
	plain := startupModel(effectNone).startupLayout()
	if plain.logo[0] == christmasLogo[0] {
		t.Fatal("the christmas logo should only appear for the christmas effect")
	}
	if plain.logoInk != nil {
		t.Fatal("the ordinary logo should carry no colour overrides")
	}

	frame := startupModel(effectSnow).renderStartupMenu()
	if !strings.Contains(frame, ansiRed+",") {
		t.Fatal("expected the hat's crown to be drawn in red")
	}
	if !strings.Contains(frame, ansiWhite+")") {
		t.Fatal("expected the fur brim to be white")
	}
	if !strings.Contains(frame, ansiWhite+"@") {
		t.Fatal("expected the bobble to be white")
	}
}

func TestChristmasHatColouring(t *testing.T) {
	l := startupModel(effectSnow).startupLayout()
	red := map[point]bool{
		{X: 3, Y: 0}: true, {X: 4, Y: 0}: true, {X: 5, Y: 0}: true, {X: 6, Y: 0}: true,
		{X: 2, Y: 1}: true, {X: 2, Y: 2}: true,
		{X: 2, Y: 3}: true, {X: 3, Y: 3}: true,
	}
	white := map[point]bool{
		{X: 5, Y: 1}: true, {X: 7, Y: 1}: true,
		{X: 4, Y: 2}: true, {X: 5, Y: 2}: true,
		{X: 2, Y: 4}: true,
	}

	for y, line := range l.logo {
		for x := range line {
			at := point{X: x, Y: y}
			switch {
			case red[at]:
				if l.logoInk[at] != ansiRed {
					t.Fatalf("hat cell (%d,%d) should be red, got %q", x, y, l.logoInk[at])
				}
			case white[at]:
				if l.logoInk[at] != ansiWhite {
					t.Fatalf("connector/bobble cell (%d,%d) should be white, got %q", x, y, l.logoInk[at])
				}
			default:
				if _, override := l.logoInk[at]; override {
					t.Fatalf("cell (%d,%d) is lettering and should keep the logo green", x, y)
				}
			}
		}
	}
}

func TestChristmasLogoKeepsTheWordmark(t *testing.T) {
	plain := startupModel(effectNone).startupLayout().logo
	xmas := startupModel(effectSnow).startupLayout().logo
	if len(xmas) != len(plain)+1 {
		t.Fatalf("expected one extra row for the hat, got %d vs %d", len(xmas), len(plain))
	}
	hatCells := map[point]bool{
		{X: 2, Y: 1}: true, {X: 5, Y: 1}: true, {X: 7, Y: 1}: true,
		{X: 2, Y: 2}: true, {X: 4, Y: 2}: true, {X: 5, Y: 2}: true,
		{X: 2, Y: 3}: true, {X: 3, Y: 3}: true,
		{X: 2, Y: 4}: true,
	}
	for i, row := range plain {
		shifted := "   " + row
		for x := range shifted {
			if hatCells[point{X: x, Y: i + 1}] {
				continue
			}
			if xmas[i+1][x] != shifted[x] {
				t.Fatalf("row %d column %d drifted: %q vs %q", i+1, x, xmas[i+1][x], shifted[x])
			}
		}
	}
}

func TestEffectsNeverCoverTheLogoOrMenu(t *testing.T) {
	for _, effect := range []effectKind{effectSnow, effectFireworks, effectFireworksUSA, effectBlood, effectConfetti} {
		m := runFrames(startupModel(effect), 200)
		if len(m.particles) == 0 {
			t.Fatalf("effect %v produced no particles", effect)
		}
		lines := strings.Split(stripANSI(m.renderStartupMenu()), "\n")
		l := m.startupLayout()
		logoX, logoY := l.logoOrigin()
		for i, row := range l.logo {
			got := string([]rune(lines[logoY+i])[logoX : logoX+len(row)])
			if got != row {
				t.Fatalf("effect %v altered logo row %d:\n got %q\nwant %q", effect, i, got, row)
			}
		}
		frame := strings.Join(lines, "\n")
		for _, want := range []string{"n: New", "o: Open", "q: Quit"} {
			if !strings.Contains(frame, want) {
				t.Fatalf("effect %v obscured %q", effect, want)
			}
		}
	}
}

func TestSnowAndConfettiFallFromTheTop(t *testing.T) {
	for _, effect := range []effectKind{effectSnow, effectConfetti} {
		m := runFrames(startupModel(effect), 40)
		top := 0
		for _, p := range m.particles {
			if p.y < 3 {
				top++
			}
		}
		if top == 0 {
			t.Fatalf("effect %v should keep seeding particles at the top of the screen", effect)
		}
	}
}

func TestBloodDripsFromTheLogo(t *testing.T) {
	m := runFrames(startupModel(effectBlood), 60)
	l := m.startupLayout()
	logoX, logoY := l.logoOrigin()
	bottom := float64(logoY + len(l.logo) - 1)
	for _, p := range m.particles {
		if p.x < float64(logoX) || p.x > float64(logoX+l.logoW) {
			t.Fatalf("blood spawned outside the logo's columns: x=%v", p.x)
		}
		if p.y < bottom {
			t.Fatalf("blood should fall from the logo's bottom edge, got y=%v want >= %v", p.y, bottom)
		}
	}
	if len(m.particles) == 0 {
		t.Fatal("expected blood drips")
	}
}

func TestFireworksRocketsBurst(t *testing.T) {
	m := startupModel(effectFireworks)
	rockets, sparks := 0, 0
	for i := 0; i < 120; i++ {
		m.advanceEffect()
		for _, p := range m.particles {
			if p.rocket {
				rockets = max(rockets, 1)
			} else {
				sparks++
			}
		}
	}
	if rockets == 0 {
		t.Fatal("expected fireworks to launch rockets")
	}
	if sparks == 0 {
		t.Fatal("expected rockets to burst into sparks")
	}
}

func TestJuly4thFireworksAreRedWhiteAndBlue(t *testing.T) {
	m := startupModel(effectFireworksUSA)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		m.advanceEffect()
		for _, p := range m.particles {
			seen[p.color] = true
		}
	}
	if len(seen) == 0 {
		t.Fatal("expected july 4th fireworks to produce particles")
	}
	for color := range seen {
		if color != ansiBright && color != ansiWhite && color != ansiBlue {
			t.Fatalf("july 4th fireworks should only be red, white or blue, got %q", color)
		}
	}
	if len(seen) < 2 {
		t.Fatalf("expected more than one of red, white and blue, got %d", len(seen))
	}
}

func TestEffectTickSpansTheWholeStartMenuFlow(t *testing.T) {
	m := startupModel(effectSnow)
	for _, mode := range []Mode{ModeStartup, ModeFileInput, ModeConfirm} {
		m.mode = mode
		if m.effectTick() == nil {
			t.Fatalf("the animation should keep running in mode %v", mode)
		}
	}
	m.mode = ModeNormal
	if m.effectTick() != nil {
		t.Fatal("the animation should stop once a chart is open")
	}
	none := startupModel(effectNone)
	if none.effectTick() != nil {
		t.Fatal("no effect should mean no ticking")
	}
}

func TestOpenDialogDoesNotFreezeTheAnimation(t *testing.T) {
	m := startupModel(effectSnow)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = out.(model)
	if m.mode != ModeFileInput {
		t.Fatalf("expected the open dialog, got %v", m.mode)
	}

	out, cmd := m.Update(effectTickMsg{})
	m = out.(model)
	if cmd == nil {
		t.Fatal("the animation stopped when the open dialog appeared")
	}
	advanced := m.effectFrame

	out, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = out.(model)
	if m.mode != ModeStartup {
		t.Fatalf("expected Esc to return to the start menu, got %v", m.mode)
	}
	out, cmd = m.Update(effectTickMsg{})
	m = out.(model)
	if cmd == nil {
		t.Fatal("the animation did not resume on returning to the start menu")
	}
	if m.effectFrame <= advanced {
		t.Fatal("expected the animation to keep advancing across the dialog")
	}
}

func TestAnimationStopsForGoodOnceAChartIsOpen(t *testing.T) {
	m := startupModel(effectSnow)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = out.(model)
	if m.mode != ModeNormal {
		t.Fatalf("expected a new chart, got %v", m.mode)
	}

	out, cmd := m.Update(effectTickMsg{})
	m = out.(model)
	if cmd != nil {
		t.Fatal("the animation should stop once a chart is created")
	}
	m.mode = ModeFileInput
	if m.effectTick() != nil {
		t.Fatal("the easter egg should not come back after a chart is open")
	}
}

func TestParticlesStayBehindTheMenuBox(t *testing.T) {
	m := runFrames(startupModel(effectConfetti), 200)
	l := m.startupLayout()
	lines := strings.Split(stripANSI(m.renderStartupMenu()), "\n")
	for y := l.boxY; y < l.boxY+l.boxH && y < len(lines); y++ {
		row := []rune(lines[y])
		for x := l.boxX; x < l.boxX+l.boxW && x < len(row); x++ {
			if strings.ContainsRune("▪▫◆●■▬", row[x]) {
				t.Fatalf("a particle showed through the menu box at (%d,%d)", x, y)
			}
		}
	}
}

func TestBloodDrawsInsideTheMenuBoxButOthersDoNot(t *testing.T) {
	at := func(m model) (model, int, int) {
		l := m.startupLayout()
		x, y := l.boxX+2, l.boxY+2+len(l.logo)
		m.particles = []particle{{x: float64(x), y: float64(y), glyph: 'O', color: ansiRed, life: 10}}
		return m, x, y
	}

	m, x, y := at(startupModel(effectBlood))
	lines := strings.Split(stripANSI(m.renderStartupMenu()), "\n")
	if got := []rune(lines[y])[x]; got != 'O' {
		t.Fatalf("expected blood drawn inside the box at (%d,%d), got %q", x, y, got)
	}

	m, x, y = at(startupModel(effectConfetti))
	lines = strings.Split(stripANSI(m.renderStartupMenu()), "\n")
	if got := []rune(lines[y])[x]; got != ' ' {
		t.Fatalf("every other effect should pass behind the box, got %q", got)
	}
}

func TestHalloweenLogoOnlyOnHalloween(t *testing.T) {
	spooky := startupModel(effectBlood).startupLayout()
	if spooky.logo[0] != halloweenLogo[0] {
		t.Fatalf("expected the halloween logo, got %q", spooky.logo[0])
	}
	for _, other := range []effectKind{effectNone, effectSnow, effectConfetti, effectFireworks} {
		if startupModel(other).startupLayout().logo[0] == halloweenLogo[0] {
			t.Fatalf("effect %v should not use the halloween logo", other)
		}
	}
}

func TestHalloweenFaceColouring(t *testing.T) {
	l := startupModel(effectBlood).startupLayout()
	const faceLo, faceHi = 11, 21

	for y, line := range l.logo {
		for x, ch := range line {
			at := point{X: x, Y: y}
			ink := l.logoInk[at]
			switch {
			case ch == ' ':
				if ink != "" {
					t.Fatalf("blank cell (%d,%d) should carry no colour", x, y)
				}
			case ch == '#':
				if ink != ansiRed {
					t.Fatalf("tooth (%d,%d) should be red, got %q", x, y, ink)
				}
			case x >= faceLo && x <= faceHi:
				if ink != ansiWhite {
					t.Fatalf("face cell (%d,%d) %q should be white, got %q", x, y, string(ch), ink)
				}
			case y == 3 && strings.ContainsRune(";,:.", ch):
				if ink != ansiRed {
					t.Fatalf("decay (%d,%d) %q should be red, got %q", x, y, string(ch), ink)
				}
			default:
				if ink != "" {
					t.Fatalf("lettering (%d,%d) %q should stay green, got %q", x, y, string(ch), ink)
				}
			}
		}
	}
}

func TestSeasonalLogosAreCentred(t *testing.T) {
	for _, effect := range []effectKind{effectNone, effectSnow, effectBlood} {
		l := startupModel(effect).startupLayout()
		lo, hi := len(l.logo[0]), 0
		for _, row := range l.logo {
			for x, ch := range row {
				if ch != ' ' {
					lo, hi = min(lo, x), max(hi, x)
				}
			}
		}
		left := lo
		right := l.contentW + 2 - (1 + hi)
		if left != right {
			t.Fatalf("effect %v logo is off centre: %d blank columns left, %d right", effect, left, right)
		}
	}
}
