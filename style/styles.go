package style

import "github.com/charmbracelet/lipgloss"

// Colour palette
var (
	colBase    = lipgloss.Color("#1e1e2e") // background
	colSurface = lipgloss.Color("#313244") // elevated surface
	colMuted   = lipgloss.Color("#6c7086") // subtle text
	colText    = lipgloss.Color("#cdd6f4") // primary text
	colSubtext = lipgloss.Color("#a6adc8") // secondary text
	colBlue    = lipgloss.Color("#89b4fa") // accent
	colGreen   = lipgloss.Color("#a6e3a1") // success
	colYellow  = lipgloss.Color("#f9e2af") // warning
	colRed     = lipgloss.Color("#f38ba8") // error / danger
	colPeach   = lipgloss.Color("#fab387") // highlight
	colMauve   = lipgloss.Color("#cba6f7") // secondary accent
)

// Shared styles
var (
	StyleTitle = lipgloss.NewStyle().
			Foreground(colBlue).
			Bold(true).
			MarginBottom(1)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(colSubtext).
			Italic(true)

	StyleSelected = lipgloss.NewStyle().
			Foreground(colBlue).
			Bold(true)

	StyleMuted = lipgloss.NewStyle().
			Foreground(colMuted)

	StyleGood = lipgloss.NewStyle().
			Foreground(colGreen)

	StyleWarn = lipgloss.NewStyle().
			Foreground(colYellow)

	StyleError = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true)

	StyleKey = lipgloss.NewStyle().
			Foreground(colMauve).
			Bold(true)

	StyleValue = lipgloss.NewStyle().
			Foreground(colText)

	StyleDanger = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colRed).
			Padding(0, 1)

	StylePrompt = lipgloss.NewStyle().
			Foreground(colBlue)

	// Sidebar
	StyleSidebar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSurface).
			Padding(1, 6).
			Width(48)

	StyleSidebarTitle = lipgloss.NewStyle().
				Foreground(colMauve).
				Bold(true).
				MarginBottom(1)

	// Main content pane
	StyleMain = lipgloss.NewStyle().
			Padding(4, 10)

	// Help bar
	StyleHelp = lipgloss.NewStyle().
			Foreground(colMuted).
			MarginTop(2)

	// Step header
	StyleStepHeader = lipgloss.NewStyle().
			Foreground(colPeach).
			Bold(true).
			MarginBottom(1)

	// Progress bar fill/empty
	StyleProgressFill  = lipgloss.NewStyle().Foreground(colBlue)
	StyleProgressEmpty = lipgloss.NewStyle().Foreground(colSurface)

	// Log viewport (phase 2)
	StyleLog = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSurface).
			Padding(0, 1)

	StyleLogLine = lipgloss.NewStyle().
			Foreground(colSubtext)

	StyleLogError = lipgloss.NewStyle().
			Foreground(colRed)

	// Checkbox
	StyleCheckboxOn  = lipgloss.NewStyle().Foreground(colGreen)
	StyleCheckboxOff = lipgloss.NewStyle().Foreground(colMuted)

	// Button
	StyleButtonActive = lipgloss.NewStyle().
				Background(colBlue).
				Foreground(colBase).
				Padding(0, 4).
				Bold(true)

	StyleButtonDanger = lipgloss.NewStyle().
				Background(colRed).
				Foreground(colBase).
				Padding(0, 4).
				Bold(true)

	StyleButtonInactive = lipgloss.NewStyle().
				Background(colSurface).
				Foreground(colMuted).
				Padding(0, 4)

	// Mode badge
	StyleModeSimple = lipgloss.NewStyle().
			Foreground(colBase).
			Background(colGreen).
			Padding(0, 1)

	StyleModeAdvanced = lipgloss.NewStyle().
				Foreground(colBase).
				Background(colPeach).
				Padding(0, 1)
)

// PlaceOverlay centers a foreground box over a background view of given total dimensions.
func PlaceOverlay(width, height int, fg, bg string) string {
	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		fg,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(colBase),
	)
}

// Checkbox renders a ✓ or ○ with colour.
func Checkbox(checked bool) string {
	if checked {
		return StyleCheckboxOn.Render("[✓]")
	}
	return StyleCheckboxOff.Render("[ ]")
}

// HelpRow renders a row of key=action pairs for the bottom help bar.
func HelpRow(pairs ...string) string {
	out := ""
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			out += StyleMuted.Render("  ·  ")
		}
		out += StyleKey.Render(pairs[i]) + StyleMuted.Render(" "+pairs[i+1])
	}
	return StyleHelp.Render(out)
}

// StyleBlue returns a lipgloss style with the blue accent colour.
// Returned as a function so callers can chain .Render() naturally.
func StyleBlue() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(colBlue).Bold(true)
}

// ToggleOn renders the active choice in an inline toggle.
// Background fill keeps it legible on both true-color and 16-color (TTY) terminals.
func ToggleOn(s string) string {
	return lipgloss.NewStyle().
		Background(colBlue).
		Foreground(colBase).
		Bold(true).
		Render(" " + s + " ")
}

// ToggleOff renders an inactive choice in an inline toggle.
func ToggleOff(s string) string {
	return StyleMuted.Render(" " + s + " ")
}

// ProgressBar renders a simple ascii progress bar of given width (0.0–1.0).
func ProgressBar(width int, pct float64) string {
	filled := int(float64(width) * pct)
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += StyleProgressFill.Render("█")
		} else {
			bar += StyleProgressEmpty.Render("░")
		}
	}
	return bar
}
