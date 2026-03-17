package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	Primary    = lipgloss.Color("#9D4EDD") // Purple - represents transformation/warp
	Secondary  = lipgloss.Color("#3B82F6") // Blue
	Success    = lipgloss.Color("#27C93F") // Green
	Warning    = lipgloss.Color("#F59E0B") // Amber
	Error      = lipgloss.Color("#EF4444") // Red
	MutedColor = lipgloss.Color("#888888") // Gray
	White      = lipgloss.Color("#FFFFFF") // White
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Secondary)

	MutedStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB"))

	HighlightStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

// Banner returns the ASCII art banner for Warp
func Banner() string {
	banner := `
 █   █ █▀▀█ █▀▀█ █▀▀█
 █▄█▄█ █▄▄█ █▄▄▀ █▄▄▀
  ▀ ▀  ▀  ▀ ▀  ▀ ▀    `
	return TitleStyle.Render(banner)
}

// Divider returns a styled divider line
func Divider() string {
	return MutedStyle.Render("──────────────────────────────────────────────")
}

// VersionLine returns the formatted version string
func VersionLine(version string) string {
	return ValueStyle.Render(" v" + version)
}

// PrintVersion prints the version
func PrintVersion(version string) {
	fmt.Println(VersionLine(version))
}

// PrintHeader prints the full header with banner, dividers, and version
func PrintHeader(version string) {
	fmt.Println()
	fmt.Println(Divider())
	fmt.Println(Banner())
	PrintVersion(version)
	fmt.Println()
	fmt.Println(Divider())
	fmt.Println()
}

// Header returns a styled section header
func Header(text string) string {
	return BoldStyle.Render("▸ " + text)
}

// PrintSuccess prints a success message with checkmark
func PrintSuccess(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(SuccessStyle.Render("✓ " + msg))
}

// PrintError prints an error message with X mark
func PrintError(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(ErrorStyle.Render("✗ " + msg))
}

// PrintWarning prints a warning message
func PrintWarning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(WarningStyle.Render("⚠ " + msg))
}

// PrintInfo prints an info message
func PrintInfo(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(InfoStyle.Render("• " + msg))
}

// PrintKeyValue prints a formatted key-value pair
func PrintKeyValue(key, value string) {
	fmt.Printf("%s: %s\n", KeyStyle.Render(key), ValueStyle.Render(value))
}

// Highlight returns highlighted text
func Highlight(s string) string {
	return HighlightStyle.Render(s)
}

// Muted returns muted/gray text
func Muted(s string) string {
	return MutedStyle.Render(s)
}

// Bold returns bold text
func Bold(s string) string {
	return BoldStyle.Render(s)
}

// Spinner runs a terminal spinner with a message until Finish is called
type Spinner struct {
	msg   string
	stop  chan struct{}
	done  sync.WaitGroup
	start time.Time
}

// NewSpinner creates and starts a spinner with the given message
func NewSpinner(msg string) *Spinner {
	s := &Spinner{
		msg:   msg,
		stop:  make(chan struct{}),
		start: time.Now(),
	}
	s.done.Add(1)
	go s.run()
	return s
}

func (s *Spinner) run() {
	defer s.done.Done()
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			elapsed := time.Since(s.start).Truncate(time.Second)
			fmt.Printf("\r\033[K  %s (%s)\n", s.msg, elapsed)
			return
		case <-ticker.C:
			elapsed := time.Since(s.start).Truncate(time.Second)
			fmt.Printf("\r\033[K  %s %s (%s)", frames[i%len(frames)], s.msg, elapsed)
			i++
		}
	}
}

// Update changes the spinner message while it's running
func (s *Spinner) Update(msg string) {
	s.msg = msg
}

// Finish stops the spinner and prints the final message with elapsed time
func (s *Spinner) Finish() {
	close(s.stop)
	s.done.Wait()
}
