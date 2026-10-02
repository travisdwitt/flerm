package tui

import (
	"math"
	"math/rand"
	"strconv"
	"time"

	cv "flerm/internal/canvas"

	tea "github.com/charmbracelet/bubbletea"
)

type effectKind int

const (
	effectNone effectKind = iota
	effectSnow
	effectFireworks
	effectBlood
	effectConfetti
	effectFireworksUSA
)

const (
	effectFrameRate = 80 * time.Millisecond
	maxParticles    = 500
)

const (
	ansiRed    = "\033[31m"
	ansiBright = "\033[91m"
	ansiWhite  = "\033[97m"
	ansiGray   = "\033[37m"
	ansiBlue   = "\033[94m"
	ansiYellow = "\033[93m"
	logoGreen  = "\033[32m"
	ansiReset  = "\033[0m"
)

var (
	confettiColors = []string{"\033[91m", "\033[92m", "\033[93m", "\033[94m", "\033[95m", "\033[96m"}
	usaColors      = []string{ansiBright, ansiWhite, ansiBlue}
)

type effectTickMsg struct{}

type particle struct {
	x, y   float64
	vx, vy float64
	glyph  rune
	color  string
	life   int
	rocket bool
}

func effectForMonthDay(month time.Month, day int) effectKind {
	switch {
	case month == time.December && day == 25:
		return effectSnow
	case month == time.January && day == 1:
		return effectFireworks
	case month == time.July && day == 4:
		return effectFireworksUSA
	case month == time.October && day == 31:
		return effectBlood
	case month == time.September && day == 17:
		return effectConfetti
	}
	return effectNone
}

func effectForDate(t time.Time) effectKind {
	return effectForMonthDay(t.Month(), t.Day())
}

func effectForOverride(mmdd string) effectKind {
	if len(mmdd) != 4 {
		return effectNone
	}
	month, err := strconv.Atoi(mmdd[:2])
	if err != nil || month < 1 || month > 12 {
		return effectNone
	}
	day, err := strconv.Atoi(mmdd[2:])
	if err != nil || day < 1 || day > 31 {
		return effectNone
	}
	return effectForMonthDay(time.Month(month), day)
}

func (m model) inStartMenuFlow() bool {
	switch m.mode {
	case ModeStartup, ModeFileInput, ModeConfirm:
		return true
	}
	return false
}

func (m model) effectTick() tea.Cmd {
	if m.effect == effectNone || !m.inStartMenuFlow() || !m.config.Particles {
		return nil
	}
	return tea.Tick(effectFrameRate, func(time.Time) tea.Msg { return effectTickMsg{} })
}

func (m *model) advanceEffect() {
	if m.effect == effectNone || m.width <= 0 || m.height <= 0 {
		return
	}
	m.effectFrame++

	var born []particle
	kept := m.particles[:0]
	for _, p := range m.particles {
		if p.rocket && p.vy >= -0.2 {
			born = append(born, burstOf(p)...)
			continue
		}
		p.x += p.vx
		p.y += p.vy
		p.vy += m.gravity()
		p.life--
		if p.life > 0 && p.y < float64(m.height) && p.x > -2 && p.x < float64(m.width)+2 {
			kept = append(kept, p)
		}
	}
	m.particles = append(kept, born...)
	if len(m.particles) < maxParticles {
		m.spawn()
	}
}

func (m model) gravity() float64 {
	switch m.effect {
	case effectFireworks, effectFireworksUSA:
		return 0.05
	case effectConfetti:
		return 0.02
	case effectBlood:
		return 0.03
	}
	return 0
}

func (m *model) spawn() {
	switch m.effect {
	case effectSnow:
		if rand.Float64() < 0.6 {
			m.particles = append(m.particles, particle{
				x: rand.Float64() * float64(m.width), y: -1,
				vx: rand.Float64()*0.3 - 0.15, vy: 0.2 + rand.Float64()*0.4,
				glyph: []rune("❄*·.")[rand.Intn(4)],
				color: []string{ansiWhite, ansiGray}[rand.Intn(2)],
				life:  m.height * 8,
			})
		}
	case effectConfetti:
		if rand.Float64() < 0.8 {
			m.particles = append(m.particles, particle{
				x: rand.Float64() * float64(m.width), y: -1,
				vx: rand.Float64()*0.6 - 0.3, vy: 0.3 + rand.Float64()*0.5,
				glyph: []rune("▪▫◆●■▬")[rand.Intn(6)],
				color: confettiColors[rand.Intn(len(confettiColors))],
				life:  m.height * 6,
			})
		}
	case effectBlood:
		m.spawnBlood()
	case effectFireworks, effectFireworksUSA:
		if m.effectFrame%14 == 0 {
			colors := m.fireworkColors()
			m.particles = append(m.particles, particle{
				x: 4 + rand.Float64()*float64(max(m.width-8, 1)), y: float64(m.height - 1),
				vy:    -1.4 - rand.Float64()*0.4,
				glyph: '|', color: colors[rand.Intn(len(colors))],
				life: m.height * 2, rocket: true,
			})
		}
	}
}

func (m model) fireworkColors() []string {
	if m.effect == effectFireworksUSA {
		return usaColors
	}
	return confettiColors
}

func (m *model) spawnBlood() {
	if rand.Float64() > 0.35 {
		return
	}
	l := m.startupLayout()
	logoX, logoY := l.logoOrigin()
	m.particles = append(m.particles, particle{
		x: float64(logoX + rand.Intn(max(l.logoW, 1))), y: float64(logoY + len(l.logo) - 1),
		vy:    0.1 + rand.Float64()*0.2,
		glyph: []rune(".,'oO")[rand.Intn(5)],
		color: []string{ansiRed, ansiBright}[rand.Intn(2)],
		life:  m.height * 8,
	})
}

func burstOf(rocket particle) []particle {
	const arms = 18
	sparks := make([]particle, 0, arms)
	for i := 0; i < arms; i++ {
		angle := float64(i) * 2 * math.Pi / arms
		speed := 0.3 + rand.Float64()*0.45
		sparks = append(sparks, particle{
			x: rocket.x, y: rocket.y,
			vx: math.Cos(angle) * speed * 1.8, vy: math.Sin(angle) * speed,
			glyph: []rune("*+·•")[rand.Intn(4)],
			color: rocket.color,
			life:  12 + rand.Intn(10),
		})
	}
	return sparks
}

var christmasLogo = []string{
	"   ,__.                            ",
	"  /  / )__ __                      ",
	"  | (_'  _|  |.-----.----.--------.",
	"  |/ |   _|  ||  -__|   _|        |",
	"  @  |__| |__||_____|__| |__|__|__|",
}

var christmasLogoInk = map[point]string{
	{X: 3, Y: 0}: ansiRed, {X: 4, Y: 0}: ansiRed, {X: 5, Y: 0}: ansiRed, {X: 6, Y: 0}: ansiRed,
	{X: 2, Y: 1}: ansiRed,
	{X: 2, Y: 2}: ansiRed,
	{X: 2, Y: 3}: ansiRed, {X: 3, Y: 3}: ansiRed,

	{X: 5, Y: 1}: ansiWhite, {X: 7, Y: 1}: ansiWhite,
	{X: 4, Y: 2}: ansiWhite, {X: 5, Y: 2}: ansiWhite,

	{X: 2, Y: 4}: ansiWhite,
}

var halloweenLogo = []string{
	"    ___ __   ,______.                ",
	"  .'  _|  |./        \\.----.--------.",
	"  |   _|  |{ (+) $ (-)|   _|        |",
	"  |;,| |,;| \\__#####_/|:;| |.;|;.|,;|",
}

var halloweenLogoInk = map[point]string{
	{X: 13, Y: 0}: ansiWhite, {X: 14, Y: 0}: ansiWhite, {X: 15, Y: 0}: ansiWhite, {X: 16, Y: 0}: ansiWhite,
	{X: 17, Y: 0}: ansiWhite, {X: 18, Y: 0}: ansiWhite, {X: 19, Y: 0}: ansiWhite, {X: 20, Y: 0}: ansiWhite,
	{X: 11, Y: 1}: ansiWhite, {X: 12, Y: 1}: ansiWhite, {X: 21, Y: 1}: ansiWhite,
	{X: 11, Y: 2}: ansiWhite, {X: 13, Y: 2}: ansiWhite, {X: 14, Y: 2}: ansiWhite, {X: 15, Y: 2}: ansiWhite,
	{X: 17, Y: 2}: ansiWhite, {X: 19, Y: 2}: ansiWhite, {X: 20, Y: 2}: ansiWhite, {X: 21, Y: 2}: ansiWhite,
	{X: 12, Y: 3}: ansiWhite, {X: 13, Y: 3}: ansiWhite, {X: 14, Y: 3}: ansiWhite,
	{X: 20, Y: 3}: ansiWhite, {X: 21, Y: 3}: ansiWhite,

	{X: 15, Y: 3}: ansiRed, {X: 16, Y: 3}: ansiRed, {X: 17, Y: 3}: ansiRed, {X: 18, Y: 3}: ansiRed,
	{X: 19, Y: 3}: ansiRed,

	{X: 3, Y: 3}: ansiRed, {X: 4, Y: 3}: ansiRed, {X: 8, Y: 3}: ansiRed, {X: 9, Y: 3}: ansiRed,
	{X: 23, Y: 3}: ansiRed, {X: 24, Y: 3}: ansiRed, {X: 28, Y: 3}: ansiRed, {X: 29, Y: 3}: ansiRed,
	{X: 31, Y: 3}: ansiRed, {X: 32, Y: 3}: ansiRed, {X: 34, Y: 3}: ansiRed, {X: 35, Y: 3}: ansiRed,
}

var newYearLogo = []string{
	"   ___ __     New year, new     ",
	" .'  _|  |.-----.----.--------. ",
	" |   _|  ||  -__|   _|        | ",
	" |__| |__||_____|__| |__|__|__| ",
}

var newYearLogoInk = map[point]string{
	{X: 14, Y: 0}: ansiWhite, {X: 15, Y: 0}: ansiWhite, {X: 16, Y: 0}: ansiWhite, {X: 17, Y: 0}: ansiWhite,
	{X: 18, Y: 0}: ansiWhite, {X: 19, Y: 0}: ansiWhite, {X: 20, Y: 0}: ansiWhite, {X: 21, Y: 0}: ansiWhite,
	{X: 22, Y: 0}: ansiWhite, {X: 23, Y: 0}: ansiWhite, {X: 24, Y: 0}: ansiWhite, {X: 25, Y: 0}: ansiWhite,
	{X: 26, Y: 0}: ansiWhite,
}

var julyFourthLogo = []string{
	"   ___  ',+'                     ",
	" .'  _ _/_`.-----.----.--------. ",
	" |   _|---||  -__|   _|:::=====| ",
	" |__| |___||_____|__| |==|==|==| ",
}

var julyFourthLogoInk = map[point]string{
	{X: 8, Y: 0}: ansiYellow, {X: 9, Y: 0}: ansiYellow, {X: 10, Y: 0}: ansiYellow,
	{X: 11, Y: 0}: ansiYellow,

	{X: 7, Y: 1}: ansiWhite, {X: 8, Y: 1}: ansiWhite, {X: 9, Y: 1}: ansiWhite,
	{X: 10, Y: 1}: ansiWhite, {X: 22, Y: 1}: ansiWhite, {X: 23, Y: 1}: ansiWhite,
	{X: 24, Y: 1}: ansiWhite, {X: 25, Y: 1}: ansiWhite, {X: 26, Y: 1}: ansiWhite,
	{X: 27, Y: 1}: ansiWhite, {X: 28, Y: 1}: ansiWhite, {X: 29, Y: 1}: ansiWhite,
	{X: 30, Y: 1}: ansiWhite, {X: 31, Y: 1}: ansiWhite, {X: 6, Y: 2}: ansiWhite, {X: 7, Y: 2}: ansiWhite, {X: 10, Y: 2}: ansiWhite,
	{X: 8, Y: 2}: ansiWhite, {X: 9, Y: 2}: ansiWhite, {X: 22, Y: 2}: ansiWhite,
	{X: 23, Y: 2}: ansiWhite, {X: 24, Y: 2}: ansiWhite, {X: 25, Y: 2}: ansiWhite,
	{X: 31, Y: 2}: ansiWhite, {X: 22, Y: 3}: ansiWhite, {X: 25, Y: 3}: ansiWhite,
	{X: 28, Y: 3}: ansiWhite, {X: 31, Y: 3}: ansiWhite,

	{X: 26, Y: 2}: ansiRed, {X: 27, Y: 2}: ansiRed, {X: 28, Y: 2}: ansiRed,
	{X: 29, Y: 2}: ansiRed, {X: 30, Y: 2}: ansiRed, {X: 6, Y: 3}: ansiRed,
	{X: 7, Y: 3}: ansiRed, {X: 8, Y: 3}: ansiRed, {X: 9, Y: 3}: ansiRed,
	{X: 10, Y: 3}: ansiRed, {X: 23, Y: 3}: ansiRed, {X: 24, Y: 3}: ansiRed,
	{X: 26, Y: 3}: ansiRed, {X: 27, Y: 3}: ansiRed, {X: 29, Y: 3}: ansiRed,
	{X: 30, Y: 3}: ansiRed,
}

func (m model) particleOverlay() map[point]string {
	if m.effect == effectNone {
		return nil
	}
	overlay := make(map[point]string, len(m.particles))
	for _, p := range m.particles {
		overlay[point{X: int(p.x), Y: int(p.y)}] = cv.Ansi(p.color) + string(p.glyph) + cv.Ansi(ansiReset)
	}
	return overlay
}
