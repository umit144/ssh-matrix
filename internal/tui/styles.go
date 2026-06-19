package tui

import "github.com/charmbracelet/lipgloss"

var (
	subtle = lipgloss.Color("#5C5C5C")
	dim    = lipgloss.Color("#808080")
	text   = lipgloss.Color("#C6C6C6")
	bright = lipgloss.Color("#EEEEEE")
	accent = lipgloss.Color("#82AAFF")

	titleStyle = lipgloss.NewStyle().
			Foreground(accent).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(dim)

	headerStyle = lipgloss.NewStyle().
			Foreground(dim).
			Bold(true).
			PaddingBottom(1)

	rowStyle = lipgloss.NewStyle().
			Foreground(text)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(bright).
				Bold(true)

	selectedIndicator = lipgloss.NewStyle().
				Foreground(accent).
				Bold(true)

	dimText = lipgloss.NewStyle().
			Foreground(subtle)

	accentText = lipgloss.NewStyle().
			Foreground(accent)

	errorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F5F"))

	containerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#333333")).
			Padding(1, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			PaddingTop(1)
)
