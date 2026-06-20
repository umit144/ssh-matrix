package ssh

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	ssh_config "github.com/kevinburke/ssh_config"
)

var userHomeDir = os.UserHomeDir

const maxIncludeDepth = 5

type wildcardHost struct {
	host     *ssh_config.Host
	settings Host
}

type configSegment struct {
	isInclude bool
	value     string
	data      []byte
}

func ParseConfig() ([]Host, error) {
	home, err := userHomeDir()
	if err != nil {
		return nil, err
	}
	return parseFile(filepath.Join(home, ".ssh", "config"), home)
}

func parseFile(path, home string) ([]Host, error) {
	return parseWithDepth(path, home, 0)
}

func parseWithDepth(path, home string, depth int) ([]Host, error) {
	if depth > maxIncludeDepth {
		return nil, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	segments := splitAtIncludes(b)

	var hosts []Host
	var wildcards []wildcardHost

	for _, seg := range segments {
		if seg.isInclude {
			included, err := resolveInclude(seg.value, home, depth)
			if err == nil {
				hosts = append(hosts, included...)
			}
			continue
		}

		cfg, err := ssh_config.DecodeBytes(seg.data)
		if err != nil {
			continue
		}

		for _, host := range cfg.Hosts {
			settings := extractSettings(host)
			names := resolvePatterns(host.Patterns)

			hasWildcard := false
			for _, name := range names {
				if isWildcard(name) {
					hasWildcard = true
				}
			}

			if hasWildcard {
				wildcards = append(wildcards, wildcardHost{host: host, settings: settings})
			}

			for _, name := range names {
				if name == "" || isWildcard(name) {
					continue
				}
				h := settings
				h.Name = name
				hosts = append(hosts, h)
			}
		}
	}

	for i := range hosts {
		for _, wc := range wildcards {
			if wc.host.Matches(hosts[i].Name) {
				mergeSettings(&hosts[i], &wc.settings)
			}
		}
	}

	applyDefaults(hosts)
	return deduplicate(hosts), nil
}

func splitAtIncludes(b []byte) []configSegment {
	var segments []configSegment
	var current bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		stripped := stripInlineComment(trimmed)
		key, val := splitKeyword(stripped)

		if strings.EqualFold(key, "include") && val != "" {
			if current.Len() > 0 {
				data := make([]byte, current.Len())
				copy(data, current.Bytes())
				segments = append(segments, configSegment{data: data})
				current.Reset()
			}
			segments = append(segments, configSegment{isInclude: true, value: val})
			continue
		}
		current.WriteString(line + "\n")
	}

	if current.Len() > 0 {
		segments = append(segments, configSegment{data: current.Bytes()})
	}

	return segments
}

func extractSettings(host *ssh_config.Host) Host {
	var h Host
	for _, node := range host.Nodes {
		kv, ok := node.(*ssh_config.KV)
		if !ok {
			continue
		}
		switch strings.ToLower(kv.Key) {
		case "hostname":
			h.HostName = kv.Value
		case "user":
			h.User = kv.Value
		case "port":
			h.Port = kv.Value
		case "identityfile":
			h.IdentityFile = kv.Value
		case "proxyjump":
			h.ProxyJump = kv.Value
		}
	}
	return h
}

// resolvePatterns reconstructs host patterns from the parsed config,
// handling quoted multi-word patterns that the package splits by space.
func resolvePatterns(patterns []*ssh_config.Pattern) []string {
	var result []string
	i := 0
	for i < len(patterns) {
		name := patterns[i].String()
		if strings.HasPrefix(name, "\"") {
			part := strings.TrimPrefix(name, "\"")
			if strings.HasSuffix(part, "\"") {
				result = append(result, strings.TrimSuffix(part, "\""))
				i++
				continue
			}
			parts := []string{part}
			i++
			for i < len(patterns) {
				part = patterns[i].String()
				if strings.HasSuffix(part, "\"") {
					parts = append(parts, strings.TrimSuffix(part, "\""))
					i++
					break
				}
				parts = append(parts, part)
				i++
			}
			result = append(result, strings.Join(parts, " "))
		} else {
			result = append(result, name)
			i++
		}
	}
	return result
}

func stripInlineComment(s string) string {
	inQuote := false
	for i, c := range s {
		if c == '"' {
			inQuote = !inQuote
		}
		if c == '#' && !inQuote {
			return strings.TrimSpace(s[:i])
		}
	}
	return s
}

func splitKeyword(line string) (string, string) {
	i := 0
	for i < len(line) && line[i] != ' ' && line[i] != '\t' && line[i] != '=' {
		i++
	}
	if i == 0 || i == len(line) {
		return line, ""
	}
	key := line[:i]
	rest := strings.TrimLeft(line[i:], " \t")
	if rest != "" && rest[0] == '=' {
		rest = strings.TrimLeft(rest[1:], " \t")
	}
	return key, rest
}

func isWildcard(pattern string) bool {
	return strings.ContainsAny(pattern, "*?!")
}

func resolveInclude(pattern, home string, depth int) ([]Host, error) {
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
		h, err := parseWithDepth(match, home, depth+1)
		if err != nil {
			continue
		}
		hosts = append(hosts, h...)
	}
	return hosts, nil
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
