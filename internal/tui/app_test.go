package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/umit/ssh-matrix/internal/ssh"
)

func testHosts() []ssh.Host {
	return []ssh.Host{
		{Name: "server-a", HostName: "10.0.0.1", User: "root", Port: "22", IdentityFile: "~/.ssh/id_rsa"},
		{Name: "server-b", HostName: "10.0.0.2", User: "deploy", Port: "2222", ProxyJump: "bastion"},
		{Name: "server-c", HostName: "10.0.0.3", User: "admin", Port: "22"},
	}
}

func sized(m Model, w, h int) Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

func press(m Model, key string) (Model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return updated.(Model), cmd
}

func pressSpecial(m Model, keyType tea.KeyType) (Model, tea.Cmd) {
	updated, cmd := m.Update(tea.KeyMsg{Type: keyType})
	return updated.(Model), cmd
}

func TestNew(t *testing.T) {
	hosts := testHosts()
	m := New(hosts)
	if len(m.hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(m.hosts))
	}
	if m.cursor != 0 {
		t.Errorf("cursor should start at 0, got %d", m.cursor)
	}
}

func TestInit(t *testing.T) {
	m := New(testHosts())
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init should return nil")
	}
}

func TestNavigateDown(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, _ = press(m, "j")
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}

	m, _ = pressSpecial(m, tea.KeyDown)
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}

	// at bottom, should not go further
	m, _ = press(m, "j")
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (clamped)", m.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	// already at top
	m, _ = press(m, "k")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}

	m, _ = press(m, "j")
	m, _ = press(m, "j")
	m, _ = pressSpecial(m, tea.KeyUp)
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestHomeEnd(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, _ = press(m, "G")
	if m.cursor != 2 {
		t.Errorf("cursor = %d, want 2", m.cursor)
	}

	m, _ = press(m, "g")
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
}

func TestQuit(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, cmd := press(m, "q")
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestEsc(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, cmd := pressSpecial(m, tea.KeyEscape)
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestCtrlC(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, cmd := pressSpecial(m, tea.KeyCtrlC)
	if !m.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

func TestEnterConnect(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, cmd := pressSpecial(m, tea.KeyEnter)
	if !m.connecting {
		t.Error("expected connecting to be true")
	}
	if cmd == nil {
		t.Error("expected exec command")
	}
}

func TestEnterEmptyList(t *testing.T) {
	m := New(nil)
	m = sized(m, 120, 40)

	m, cmd := pressSpecial(m, tea.KeyEnter)
	if m.connecting {
		t.Error("should not connect with empty hosts")
	}
	if cmd != nil {
		t.Error("should return nil cmd for empty list")
	}
}

func TestKeysBlockedWhileConnecting(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.connecting = true

	m, _ = press(m, "j")
	if m.cursor != 0 {
		t.Error("cursor should not move while connecting")
	}
}

func TestSSHFinishedSuccess(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.connecting = true

	updated, cmd := m.Update(sshFinishedMsg{err: nil})
	m = updated.(Model)

	if m.connecting {
		t.Error("connecting should be false after finish")
	}
	if m.status != "disconnected" {
		t.Errorf("status = %q, want %q", m.status, "disconnected")
	}
	if m.statusErr {
		t.Error("statusErr should be false on success")
	}
	if cmd == nil {
		t.Error("expected clear timer command")
	}
}

func TestSSHFinishedError(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.connecting = true

	updated, cmd := m.Update(sshFinishedMsg{err: errors.New("connection refused")})
	m = updated.(Model)

	if m.connecting {
		t.Error("connecting should be false")
	}
	if m.status != "connection refused" {
		t.Errorf("status = %q, want %q", m.status, "connection refused")
	}
	if !m.statusErr {
		t.Error("statusErr should be true")
	}
	if cmd == nil {
		t.Error("expected clear timer command")
	}
}

func TestClearStatusMsg(t *testing.T) {
	m := New(testHosts())
	m.status = "some error"
	m.statusErr = true

	updated, _ := m.Update(clearStatusMsg{})
	m = updated.(Model)

	if m.status != "" {
		t.Error("status should be cleared")
	}
	if m.statusErr {
		t.Error("statusErr should be cleared")
	}
}

func TestWindowResize(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	if m.width != 120 || m.height != 40 {
		t.Errorf("size = %dx%d, want 120x40", m.width, m.height)
	}
}

func TestFriendlyErrors(t *testing.T) {
	cases := []struct {
		err  string
		want string
	}{
		{"ssh: connection refused", "connection refused"},
		{"dial tcp: timed out", "connection timed out"},
		{"connect: timeout", "connection timed out"},
		{"connect: no route to host", "no route to host"},
		{"connect: network is unreachable", "network unreachable"},
		{"permission denied (publickey)", "permission denied"},
		{"host key verification failed", "host key verification failed"},
		{"exit status 255", "ssh connection failed"},
		{"exit status 1", "session ended with exit status 1"},
		{"something unexpected", "something unexpected"},
	}
	for _, tc := range cases {
		got := friendlyError(errors.New(tc.err))
		if got != tc.want {
			t.Errorf("friendlyError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestViewQuitting(t *testing.T) {
	m := New(testHosts())
	m.quitting = true
	if m.View() != "" {
		t.Error("quitting view should be empty")
	}
}

func TestViewZeroWidth(t *testing.T) {
	m := New(testHosts())
	if m.View() != "" {
		t.Error("zero-width view should be empty")
	}
}

func TestViewEmptyHosts(t *testing.T) {
	m := New(nil)
	m = sized(m, 120, 40)

	v := m.View()
	if !strings.Contains(v, "no hosts found") {
		t.Error("empty host view should show 'no hosts found'")
	}
	if !strings.Contains(v, "ssh-matrix") {
		t.Error("should still show header")
	}
}

func TestViewNormal(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	v := m.View()
	if !strings.Contains(v, "ssh-matrix") {
		t.Error("should show header")
	}
	if !strings.Contains(v, "server-a") {
		t.Error("should show first host")
	}
	if !strings.Contains(v, "10.0.0.1") {
		t.Error("should show hostname")
	}
	if !strings.Contains(v, "3 hosts") {
		t.Error("should show host count")
	}
	if !strings.Contains(v, "key: ~/.ssh/id_rsa") {
		t.Error("should show identity file for selected host")
	}
}

func TestViewSelectedWithProxy(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, _ = press(m, "j") // move to server-b which has ProxyJump
	v := m.View()
	if !strings.Contains(v, "via: bastion") {
		t.Error("should show proxy jump for selected host")
	}
}

func TestViewNoDetailWhenEmpty(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	m, _ = press(m, "j")
	m, _ = press(m, "j") // server-c has no IdentityFile or ProxyJump
	v := m.View()
	if strings.Contains(v, "key:") {
		t.Error("should not show 'key:' when IdentityFile is empty")
	}
}

func TestViewWithErrorStatus(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.status = "connection refused"
	m.statusErr = true

	v := m.View()
	if !strings.Contains(v, "connection refused") {
		t.Error("should show error status")
	}
}

func TestViewWithSuccessStatus(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.status = "disconnected"
	m.statusErr = false

	v := m.View()
	if !strings.Contains(v, "disconnected") {
		t.Error("should show disconnected status")
	}
}

func TestScrolling(t *testing.T) {
	var hosts []ssh.Host
	for i := 0; i < 50; i++ {
		hosts = append(hosts, ssh.Host{
			Name:     strings.Repeat("h", 3) + string(rune('a'+i%26)),
			HostName: "10.0.0.1",
			User:     "root",
			Port:     "22",
		})
	}

	m := New(hosts)
	m = sized(m, 120, 30) // small terminal to force scrolling

	// move to bottom
	for i := 0; i < 49; i++ {
		m, _ = press(m, "j")
	}
	if m.cursor != 49 {
		t.Errorf("cursor = %d, want 49", m.cursor)
	}

	v := m.View()
	if !strings.Contains(v, "more") {
		t.Error("should show scroll indicator")
	}

	// move back to top
	m, _ = press(m, "g")
	if m.cursor != 0 || m.offset != 0 {
		t.Errorf("cursor=%d offset=%d, want 0,0", m.cursor, m.offset)
	}
}

func TestVisibleRowsMinimum(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 5) // very small terminal

	vis := m.visibleRows()
	if vis < 1 {
		t.Errorf("visibleRows = %d, should be at least 1", vis)
	}
}

func TestVisibleRowsWithStatus(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)

	without := m.visibleRows()
	m.status = "error"
	with := m.visibleRows()

	if with != without-1 {
		t.Errorf("status should reduce visible rows by 1: without=%d, with=%d", without, with)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		max  int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 5, "hell…"},
		{"ab", 1, "a"},
		{"", 5, ""},
		{"日本語テスト", 4, "日本語…"},
	}
	for _, tc := range cases {
		got := truncate(tc.s, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.s, tc.max, got, tc.want)
		}
	}
}

func TestConnectSSH(t *testing.T) {
	m := New(testHosts())
	cmd := m.connectSSH()
	if cmd == nil {
		t.Error("connectSSH should return a command")
	}
}

func TestSSHExecCallback(t *testing.T) {
	msg := sshExecCallback(nil)
	if m, ok := msg.(sshFinishedMsg); !ok || m.err != nil {
		t.Error("expected sshFinishedMsg with nil error")
	}

	err := errors.New("fail")
	msg = sshExecCallback(err)
	if m, ok := msg.(sshFinishedMsg); !ok || m.err != err {
		t.Error("expected sshFinishedMsg with error")
	}
}

func TestClearStatusAfter(t *testing.T) {
	cmd := clearStatusAfter(0)
	if cmd == nil {
		t.Error("clearStatusAfter should return a command")
	}
}

func TestClearStatusTick(t *testing.T) {
	msg := clearStatusTick(time.Time{})
	if _, ok := msg.(clearStatusMsg); !ok {
		t.Error("expected clearStatusMsg")
	}
}

func TestScrollUpIndicator(t *testing.T) {
	var hosts []ssh.Host
	for i := 0; i < 50; i++ {
		hosts = append(hosts, ssh.Host{
			Name:     fmt.Sprintf("host-%02d", i),
			HostName: "10.0.0.1",
			User:     "root",
			Port:     "22",
		})
	}

	m := New(hosts)
	m = sized(m, 120, 30)

	// scroll down enough to create an offset > 0
	for i := 0; i < 20; i++ {
		m, _ = press(m, "j")
	}

	v := m.View()
	if !strings.Contains(v, "↑") {
		t.Error("should show up-scroll indicator when offset > 0")
	}
	if !strings.Contains(v, "↓") {
		t.Error("should show down-scroll indicator when more items below")
	}
}

func TestClampScrollNegativeOffset(t *testing.T) {
	m := New(testHosts())
	m = sized(m, 120, 40)
	m.offset = -5
	m.clampScroll()
	if m.offset != 0 {
		t.Errorf("offset = %d, want 0", m.offset)
	}
}

func TestViewScrolledNoDownIndicator(t *testing.T) {
	var hosts []ssh.Host
	for i := 0; i < 5; i++ {
		hosts = append(hosts, ssh.Host{
			Name:     fmt.Sprintf("host-%d", i),
			HostName: "10.0.0.1",
			User:     "root",
			Port:     "22",
		})
	}

	m := New(hosts)
	m = sized(m, 120, 40) // large enough to show all

	v := m.View()
	if strings.Contains(v, "more") {
		t.Error("should not show scroll indicators when all items fit")
	}
}
