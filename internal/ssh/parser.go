package ssh

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

var userHomeDir = os.UserHomeDir

type wildcardBlock struct {
	patterns []string
	settings Host
}

func ParseConfig() ([]Host, error) {
	home, err := userHomeDir()
	if err != nil {
		return nil, err
	}
	return parseFile(filepath.Join(home, ".ssh", "config"), home)
}

func parseFile(path, home string) ([]Host, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var hosts []Host
	var wildcards []wildcardBlock
	var pending []*Host

	flush := func() {
		var wcPatterns []string
		for _, h := range pending {
			if isWildcard(h.Name) {
				wcPatterns = append(wcPatterns, h.Name)
				continue
			}
			hosts = append(hosts, *h)
		}
		if len(wcPatterns) > 0 && len(pending) > 0 {
			wildcards = append(wildcards, wildcardBlock{
				patterns: wcPatterns,
				settings: *pending[0],
			})
		}
		pending = nil
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := stripComment(scanner.Text())
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		key, val := splitDirective(line)
		if key == "" {
			continue
		}

		switch strings.ToLower(key) {
		case "host":
			flush()
			for _, pattern := range splitHostPatterns(val) {
				pending = append(pending, &Host{Name: pattern})
			}
		case "match":
			flush()
		case "include":
			flush()
			included, err := resolveInclude(val, home)
			if err == nil {
				hosts = append(hosts, included...)
			}
		case "hostname":
			for _, h := range pending {
				h.HostName = unquote(val)
			}
		case "user":
			for _, h := range pending {
				h.User = unquote(val)
			}
		case "port":
			for _, h := range pending {
				h.Port = unquote(val)
			}
		case "identityfile":
			for _, h := range pending {
				h.IdentityFile = unquote(val)
			}
		case "proxyjump":
			for _, h := range pending {
				h.ProxyJump = unquote(val)
			}
		}
	}

	flush()
	applyWildcards(hosts, wildcards)
	applyDefaults(hosts)
	return deduplicate(hosts), scanner.Err()
}

func deduplicate(hosts []Host) []Host {
	seen := make(map[string]struct{}, len(hosts))
	out := make([]Host, 0, len(hosts))
	for _, h := range hosts {
		if _, ok := seen[h.Name]; ok {
			continue
		}
		seen[h.Name] = struct{}{}
		out = append(out, h)
	}
	return out
}

func splitDirective(line string) (string, string) {
	key, rest := splitFirstToken(line)
	if rest != "" {
		return key, rest
	}
	if idx := strings.IndexByte(line, '='); idx != -1 {
		k := strings.TrimSpace(line[:idx])
		v := strings.TrimSpace(line[idx+1:])
		if k != "" {
			return k, v
		}
	}
	return key, ""
}

func splitFirstToken(line string) (string, string) {
	i := 0
	for i < len(line) && line[i] != ' ' && line[i] != '\t' && line[i] != '=' {
		i++
	}
	if i == len(line) {
		return line, ""
	}
	key := line[:i]
	rest := strings.TrimLeft(line[i:], " \t")
	if rest != "" && rest[0] == '=' {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	return key, rest
}

func stripComment(line string) string {
	inQuote := false
	for i, c := range line {
		if c == '"' {
			inQuote = !inQuote
		}
		if c == '#' && !inQuote {
			return line[:i]
		}
	}
	return line
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func isWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?!")
}

func splitHostPatterns(val string) []string {
	var patterns []string
	s := strings.TrimSpace(val)
	for s != "" {
		if s[0] == '"' {
			end := strings.IndexByte(s[1:], '"')
			if end == -1 {
				patterns = append(patterns, s[1:])
				break
			}
			patterns = append(patterns, s[1:end+1])
			s = strings.TrimSpace(s[end+2:])
		} else {
			end := strings.IndexAny(s, " \t")
			if end == -1 {
				patterns = append(patterns, s)
				break
			}
			patterns = append(patterns, s[:end])
			s = strings.TrimSpace(s[end:])
		}
	}
	if len(patterns) == 0 {
		return []string{val}
	}
	return patterns
}

func resolveInclude(pattern, home string) ([]Host, error) {
	if strings.HasPrefix(pattern, "~/") {
		pattern = filepath.Join(home, pattern[2:])
	} else if pattern == "~" {
		pattern = home
	}

	if !filepath.IsAbs(pattern) {
		pattern = filepath.Join(home, ".ssh", pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var hosts []Host
	for _, match := range matches {
		h, err := parseFile(match, home)
		if err != nil {
			continue
		}
		hosts = append(hosts, h...)
	}
	return hosts, nil
}

func applyWildcards(hosts []Host, wildcards []wildcardBlock) {
	for i := range hosts {
		for _, wc := range wildcards {
			if matchesPatterns(hosts[i].Name, wc.patterns) {
				mergeSettings(&hosts[i], &wc.settings)
			}
		}
	}
}

func applyDefaults(hosts []Host) {
	for i := range hosts {
		if hosts[i].HostName == "" {
			hosts[i].HostName = hosts[i].Name
		}
		if hosts[i].Port == "" {
			hosts[i].Port = "22"
		}
	}
}

func matchesPatterns(name string, patterns []string) bool {
	matched := false
	for _, p := range patterns {
		if strings.HasPrefix(p, "!") {
			if ok, _ := filepath.Match(p[1:], name); ok {
				return false
			}
		} else {
			if ok, _ := filepath.Match(p, name); ok {
				matched = true
			}
		}
	}
	return matched
}

func mergeSettings(h, defaults *Host) {
	if h.HostName == "" && defaults.HostName != "" {
		h.HostName = defaults.HostName
	}
	if h.User == "" && defaults.User != "" {
		h.User = defaults.User
	}
	if h.Port == "" && defaults.Port != "" {
		h.Port = defaults.Port
	}
	if h.IdentityFile == "" && defaults.IdentityFile != "" {
		h.IdentityFile = defaults.IdentityFile
	}
	if h.ProxyJump == "" && defaults.ProxyJump != "" {
		h.ProxyJump = defaults.ProxyJump
	}
}
