package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umit144/ssh-matrix/internal/ssh"
)

const chromeHeight = 16

type sshFinishedMsg struct{ err error }
type clearStatusMsg struct{}

type Model struct {
	hosts      []ssh.Host
	cursor     int
	offset     int
	width      int
	height     int
	quitting   bool
	connecting bool
	status     string
	statusErr  bool
}

func New(hosts []ssh.Host) Model {
	return Model{hosts: hosts}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.connecting {
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampScroll()
			}
		case "down", "j":
			if m.cursor < len(m.hosts)-1 {
				m.cursor++
				m.clampScroll()
			}
		case "home", "g":
			m.cursor = 0
			m.clampScroll()
		case "end", "G":
			m.cursor = len(m.hosts) - 1
			m.clampScroll()
		case "enter":
			if len(m.hosts) == 0 {
				return m, nil
			}
			m.connecting = true
			m.status = ""
			return m, m.connectSSH()
		}

	case sshFinishedMsg:
		m.connecting = false
		if msg.err != nil {
			m.statusErr = true
			m.status = friendlyError(msg.err)
			return m, clearStatusAfter(5 * time.Second)
		}
		m.statusErr = false
		m.status = "disconnected"
		return m, clearStatusAfter(3 * time.Second)

	case clearStatusMsg:
		m.status = ""
		m.statusErr = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	}
	return m, nil
}

func (m *Model) visibleRows() int {
	available := m.height - chromeHeight
	if m.status != "" {
		available--
	}
	if available < 1 {
		available = 1
	}
	return available
}

func (m *Model) clampScroll() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) connectSSH() tea.Cmd {
	host := m.hosts[m.cursor]
	c := exec.Command("ssh", host.Name)
	return tea.ExecProcess(c, sshExecCallback)
}

func sshExecCallback(err error) tea.Msg {
	return sshFinishedMsg{err: err}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, clearStatusTick)
}

func clearStatusTick(time.Time) tea.Msg {
	return clearStatusMsg{}
}

func friendlyError(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "connection refused"):
		return "connection refused"
	case strings.Contains(s, "timed out") || strings.Contains(s, "timeout"):
		return "connection timed out"
	case strings.Contains(s, "no route to host"):
		return "no route to host"
	case strings.Contains(s, "network is unreachable"):
		return "network unreachable"
	case strings.Contains(s, "permission denied"):
		return "permission denied"
	case strings.Contains(s, "host key verification failed"):
		return "host key verification failed"
	case strings.Contains(s, "exit status 255"):
		return "ssh connection failed"
	case strings.Contains(s, "exit status"):
		return "session ended with " + s
	default:
		return s
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return ""
	}

	logo := titleStyle.Render("ssh-matrix")
	tagline := subtitleStyle.Render("your hosts, one keystroke away")
	header := lipgloss.JoinVertical(lipgloss.Center, logo, tagline)

	if len(m.hosts) == 0 {
		empty := containerStyle.Render(
			dimText.Render("  no hosts found in ~/.ssh/config"),
		)
		help := helpStyle.Render("  q quit")
		full := lipgloss.JoinVertical(lipgloss.Center, "", header, "", empty, help)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
	}

	nameW := 20
	hostW := 18
	userW := 12
	portW := 6

	tableHeader := headerStyle.Render(
		fmt.Sprintf("  %-*s  %-*s  %-*s  %*s",
			nameW, "HOST",
			hostW, "ADDRESS",
			userW, "USER",
			portW, "PORT",
		),
	)

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.hosts) {
		end = len(m.hosts)
	}

	var rows []string

	if m.offset > 0 {
		rows = append(rows, dimText.Render(fmt.Sprintf("  ↑ %d more", m.offset)))
	}

	for i := m.offset; i < end; i++ {
		h := m.hosts[i]
		indicator := "  "
		name := rowStyle.Render(fmt.Sprintf("%-*s", nameW, truncate(h.Name, nameW)))
		host := dimText.Render(fmt.Sprintf("%-*s", hostW, truncate(h.HostName, hostW)))
		user := dimText.Render(fmt.Sprintf("%-*s", userW, truncate(h.User, userW)))
		port := dimText.Render(fmt.Sprintf("%*s", portW, h.Port))

		if i == m.cursor {
			indicator = selectedIndicator.Render("▸ ")
			name = selectedRowStyle.Render(fmt.Sprintf("%-*s", nameW, truncate(h.Name, nameW)))
			host = accentText.Render(fmt.Sprintf("%-*s", hostW, truncate(h.HostName, hostW)))
			user = accentText.Render(fmt.Sprintf("%-*s", userW, truncate(h.User, userW)))
			port = accentText.Render(fmt.Sprintf("%*s", portW, h.Port))
		}

		row := fmt.Sprintf("%s%s  %s  %s  %s", indicator, name, host, user, port)
		rows = append(rows, row)
	}

	if end < len(m.hosts) {
		rows = append(rows, dimText.Render(fmt.Sprintf("  ↓ %d more", len(m.hosts)-end)))
	}

	table := lipgloss.JoinVertical(lipgloss.Left, append([]string{tableHeader}, rows...)...)

	selected := m.hosts[m.cursor]
	var detailParts []string
	if selected.IdentityFile != "" {
		detailParts = append(detailParts, "key: "+selected.IdentityFile)
	}
	if selected.ProxyJump != "" {
		detailParts = append(detailParts, "via: "+selected.ProxyJump)
	}
	detail := dimText.Render("  " + strings.Join(detailParts, "  ·  "))

	var statusLine string
	if m.status != "" {
		if m.statusErr {
			statusLine = errorText.Render("  ✕ " + m.status)
		} else {
			statusLine = dimText.Render("  " + m.status)
		}
	}

	contentParts := []string{table, "", detail}
	if statusLine != "" {
		contentParts = append(contentParts, statusLine)
	}

	content := containerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left, contentParts...),
	)

	count := dimText.Render(fmt.Sprintf("  %d hosts", len(m.hosts)))
	help := helpStyle.Render("  ↑↓ navigate  enter connect  / filter  q quit")

	full := lipgloss.JoinVertical(lipgloss.Center,
		"",
		header,
		"",
		content,
		lipgloss.JoinHorizontal(lipgloss.Top, help, "    ", count),
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, full)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}
