package ssh

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	dir := t.TempDir()
	config := `
# Global defaults
Host *
    ServerAliveInterval 60

# Production servers
Host production-web
    HostName 192.168.1.10
    User deploy
    Port 22
    IdentityFile ~/.ssh/prod_rsa

Host staging-api
    HostName=10.0.0.50
    User = admin

# Quoted values
Host "dev server"
    HostName 172.16.0.5
    User "root"
    IdentityFile "~/.ssh/my key"

# Inline comment
Host jump-host  # this is the bastion
    HostName 203.0.113.1
    User bastion
    Port 2222
    ProxyJump none

# Multi-pattern host line — all three should appear
Host web1 web2 web3
    User deploy
    Port 22

# Wildcard — should be skipped
Host *.internal
    User internal

# Match block — should not break parsing
Match host *.example.com
    User matched

Host after-match
    HostName 10.0.0.1
    User final
`

	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	expected := []struct {
		name     string
		hostName string
		user     string
		port     string
		proxy    string
		key      string
	}{
		{"production-web", "192.168.1.10", "deploy", "22", "", "~/.ssh/prod_rsa"},
		{"staging-api", "10.0.0.50", "admin", "22", "", ""},
		{"dev server", "172.16.0.5", "root", "22", "", "~/.ssh/my key"},
		{"jump-host", "203.0.113.1", "bastion", "2222", "none", ""},
		{"web1", "web1", "deploy", "22", "", ""},
		{"web2", "web2", "deploy", "22", "", ""},
		{"web3", "web3", "deploy", "22", "", ""},
		{"after-match", "10.0.0.1", "final", "22", "", ""},
	}

	if len(hosts) != len(expected) {
		t.Fatalf("got %d hosts, want %d\nhosts: %+v", len(hosts), len(expected), hosts)
	}

	for i, want := range expected {
		got := hosts[i]
		if got.Name != want.name {
			t.Errorf("host[%d].Name = %q, want %q", i, got.Name, want.name)
		}
		if got.HostName != want.hostName {
			t.Errorf("host[%d].HostName = %q, want %q", i, got.HostName, want.hostName)
		}
		if got.User != want.user {
			t.Errorf("host[%d].User = %q, want %q", i, got.User, want.user)
		}
		if got.Port != want.port {
			t.Errorf("host[%d].Port = %q, want %q", i, got.Port, want.port)
		}
		if got.ProxyJump != want.proxy {
			t.Errorf("host[%d].ProxyJump = %q, want %q", i, got.ProxyJump, want.proxy)
		}
		if got.IdentityFile != want.key {
			t.Errorf("host[%d].IdentityFile = %q, want %q", i, got.IdentityFile, want.key)
		}
	}
}

func TestParseInclude(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	os.MkdirAll(filepath.Join(sshDir, "config.d"), 0o755)

	extra := `
Host included-host
    HostName 10.10.10.10
    User included
`
	os.WriteFile(filepath.Join(sshDir, "config.d", "extra.conf"), []byte(extra), 0o644)

	main := `
Include config.d/*.conf

Host main-host
    HostName 10.0.0.1
    User main
`
	path := filepath.Join(sshDir, "config")
	os.WriteFile(path, []byte(main), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2\nhosts: %+v", len(hosts), hosts)
	}

	if hosts[0].Name != "included-host" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "included-host")
	}
	if hosts[1].Name != "main-host" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "main-host")
	}
}

func TestDeduplication(t *testing.T) {
	dir := t.TempDir()
	config := `
Host myserver
    HostName 10.0.0.1
    User first

Host myserver
    HostName 10.0.0.2
    User second

Host other
    HostName 10.0.0.3
`

	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2\nhosts: %+v", len(hosts), hosts)
	}

	if hosts[0].Name != "myserver" || hosts[0].HostName != "10.0.0.1" {
		t.Errorf("first occurrence should win: got %+v", hosts[0])
	}
	if hosts[1].Name != "other" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "other")
	}
}

func TestEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte("# just comments\n\n"), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("got %d hosts, want 0", len(hosts))
	}
}

func TestTildeInclude(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	os.MkdirAll(sshDir, 0o755)

	extra := `
Host tilde-host
    HostName 10.0.0.5
`
	os.WriteFile(filepath.Join(sshDir, "extra"), []byte(extra), 0o644)

	main := `
Include ~/.ssh/extra
`
	path := filepath.Join(sshDir, "config")
	os.WriteFile(path, []byte(main), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 1 || hosts[0].Name != "tilde-host" {
		t.Fatalf("expected tilde-host, got %+v", hosts)
	}
}

func TestFileNotFound(t *testing.T) {
	_, err := parseFile("/nonexistent/path/config", "/tmp")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestIncludeNoMatch(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	os.MkdirAll(sshDir, 0o755)

	config := `
Include config.d/*.conf

Host only-host
    HostName 10.0.0.1
`
	path := filepath.Join(sshDir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Name != "only-host" {
		t.Fatalf("expected only-host, got %+v", hosts)
	}
}

func TestIncludeBrokenFile(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	confDir := filepath.Join(sshDir, "config.d")
	os.MkdirAll(confDir, 0o755)

	// create a directory where a file is expected — parseFile will fail on it
	os.MkdirAll(filepath.Join(confDir, "bad.conf"), 0o755)

	good := `
Host good-host
    HostName 10.0.0.2
`
	os.WriteFile(filepath.Join(confDir, "good.conf"), []byte(good), 0o644)

	config := `
Include config.d/*.conf

Host main-host
    HostName 10.0.0.1
`
	path := filepath.Join(sshDir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2\nhosts: %+v", len(hosts), hosts)
	}
}

func TestSplitDirectiveKeyOnly(t *testing.T) {
	key, val := splitDirective("SomeKeyword")
	if key != "SomeKeyword" || val != "" {
		t.Errorf("splitDirective(%q) = (%q, %q), want (%q, %q)", "SomeKeyword", key, val, "SomeKeyword", "")
	}
}

func TestSplitHostPatternsEmpty(t *testing.T) {
	patterns := splitHostPatterns("")
	if len(patterns) != 1 || patterns[0] != "" {
		t.Errorf("splitHostPatterns(\"\") = %v, want [\"\"]", patterns)
	}
}

func TestSplitHostPatternsUnterminatedQuote(t *testing.T) {
	patterns := splitHostPatterns(`"unterminated`)
	if len(patterns) != 1 || patterns[0] != "unterminated" {
		t.Errorf("got %v, want [\"unterminated\"]", patterns)
	}
}

func TestIncludeBareHome(t *testing.T) {
	hosts, _ := resolveInclude("~", t.TempDir())
	if hosts != nil {
		t.Errorf("expected nil hosts for bare ~, got %+v", hosts)
	}
}

func TestParseConfigPublic(t *testing.T) {
	_, _ = ParseConfig()
}

func TestParseConfigHomeDirError(t *testing.T) {
	orig := userHomeDir
	userHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}
	defer func() { userHomeDir = orig }()

	_, err := ParseConfig()
	if err == nil || err.Error() != "no home" {
		t.Errorf("expected 'no home' error, got %v", err)
	}
}

func TestSplitDirectiveEqualsNoValue(t *testing.T) {
	key, val := splitDirective("Key=")
	if key != "Key" {
		t.Errorf("key = %q, want %q", key, "Key")
	}
	if val != "" {
		t.Errorf("val = %q, want empty", val)
	}
}

func TestSplitDirectiveBareEquals(t *testing.T) {
	key, val := splitDirective("=")
	if key != "" || val != "" {
		t.Errorf("splitDirective(\"=\") = (%q, %q), want (\"\", \"\")", key, val)
	}
}

func TestIncludeInvalidGlob(t *testing.T) {
	hosts, err := resolveInclude("[", t.TempDir())
	if err == nil {
		t.Error("expected error for invalid glob pattern")
	}
	if hosts != nil {
		t.Error("expected nil hosts")
	}
}

func TestMalformedLineSkipped(t *testing.T) {
	dir := t.TempDir()
	config := "Host myhost\n    HostName 10.0.0.1\n=\n"
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Name != "myhost" {
		t.Fatalf("expected myhost, got %+v", hosts)
	}
}

func TestEqualsOnlyDirectiveInConfig(t *testing.T) {
	dir := t.TempDir()
	config := `
Host myhost
    HostName=10.0.0.1
    User=deploy
    Port=
`
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(hosts))
	}
	if hosts[0].HostName != "10.0.0.1" {
		t.Errorf("HostName = %q, want %q", hosts[0].HostName, "10.0.0.1")
	}
	if hosts[0].User != "deploy" {
		t.Errorf("User = %q, want %q", hosts[0].User, "deploy")
	}
}

func TestWildcardSettings(t *testing.T) {
	dir := t.TempDir()
	config := `
Host server-a
    HostName 10.0.0.1

Host server-b
    HostName 10.0.0.2
    User admin

Host other
    HostName 10.0.0.3

Host server-*
    User root
    Port 2222
    IdentityFile ~/.ssh/server_key
    ProxyJump bastion
`
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 3 {
		t.Fatalf("got %d hosts, want 3\nhosts: %+v", len(hosts), hosts)
	}

	if hosts[0].User != "root" {
		t.Errorf("server-a User = %q, want %q", hosts[0].User, "root")
	}
	if hosts[0].Port != "2222" {
		t.Errorf("server-a Port = %q, want %q", hosts[0].Port, "2222")
	}
	if hosts[0].IdentityFile != "~/.ssh/server_key" {
		t.Errorf("server-a IdentityFile = %q, want %q", hosts[0].IdentityFile, "~/.ssh/server_key")
	}
	if hosts[0].ProxyJump != "bastion" {
		t.Errorf("server-a ProxyJump = %q, want %q", hosts[0].ProxyJump, "bastion")
	}

	if hosts[1].User != "admin" {
		t.Errorf("server-b User = %q, want %q (explicit should win)", hosts[1].User, "admin")
	}
	if hosts[1].Port != "2222" {
		t.Errorf("server-b Port = %q, want %q", hosts[1].Port, "2222")
	}

	if hosts[2].User != "" {
		t.Errorf("other User = %q, want empty (should not match server-*)", hosts[2].User)
	}
	if hosts[2].Port != "22" {
		t.Errorf("other Port = %q, want %q", hosts[2].Port, "22")
	}
}

func TestWildcardNegation(t *testing.T) {
	dir := t.TempDir()
	config := `
Host server-a
    HostName 10.0.0.1

Host server-b
    HostName 10.0.0.2

Host * !server-b
    User global
`
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("got %d hosts, want 2\nhosts: %+v", len(hosts), hosts)
	}

	if hosts[0].User != "global" {
		t.Errorf("server-a User = %q, want %q", hosts[0].User, "global")
	}
	if hosts[1].User != "" {
		t.Errorf("server-b User = %q, want empty (negated)", hosts[1].User)
	}
}

func TestWildcardHostName(t *testing.T) {
	dir := t.TempDir()
	config := `
Host myhost
    User deploy

Host my*
    HostName 10.0.0.1
`
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(hosts))
	}
	if hosts[0].HostName != "10.0.0.1" {
		t.Errorf("HostName = %q, want %q", hosts[0].HostName, "10.0.0.1")
	}
}

func TestWildcardNoMatch(t *testing.T) {
	dir := t.TempDir()
	config := `
Host myhost
    HostName 10.0.0.1

Host other-*
    User root
`
	path := filepath.Join(dir, "config")
	os.WriteFile(path, []byte(config), 0o644)

	hosts, err := parseFile(path, dir)
	if err != nil {
		t.Fatalf("parseFile error: %v", err)
	}

	if len(hosts) != 1 {
		t.Fatalf("got %d hosts, want 1", len(hosts))
	}
	if hosts[0].User != "" {
		t.Errorf("User = %q, want empty", hosts[0].User)
	}
}
