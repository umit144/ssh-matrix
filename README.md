# ssh-matrix

[![CI](https://github.com/umit144/ssh-matrix/actions/workflows/ci.yml/badge.svg)](https://github.com/umit144/ssh-matrix/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A terminal UI for browsing and connecting to SSH hosts from your `~/.ssh/config`.

```
                        ssh-matrix
             your hosts, one keystroke away

  ╭──────────────────────────────────────────────────────╮
  │  HOST                ADDRESS          USER      PORT │
  │                                                      │
  │  ▸ production-web    192.168.1.10     deploy      22 │
  │    staging-api       10.0.0.50        admin       22 │
  │    jump-host         203.0.113.1      bastion   2222 │
  │    dev-server        172.16.0.5       root        22 │
  │                                                      │
  │  key: ~/.ssh/prod_rsa                                │
  ╰──────────────────────────────────────────────────────╯
    ↑↓ navigate  enter connect  / filter  q quit  4 hosts
```

## Features

- Reads `~/.ssh/config` with full support for `Include`, `Match`, multi-pattern `Host`, quoted values, and inline comments
- Vim-style keyboard navigation (`j`/`k`, `g`/`G`)
- One-keystroke SSH connections via `ssh` subprocess
- Friendly error messages for common connection failures
- Automatic deduplication and wildcard filtering
- Cross-platform: macOS and Linux

## Install

**Homebrew**

```sh
brew install umit144/tap/ssh-matrix
```

On macOS, you may need to remove the quarantine attribute after installation:

```sh
sudo xattr -r -d com.apple.quarantine $(which ssh-matrix)
```

**Go**

```sh
go install github.com/umit144/ssh-matrix@latest
```

**Binary** — download from [Releases](https://github.com/umit144/ssh-matrix/releases/latest)

## Keybindings

| Key              | Action                    |
| ---------------- | ------------------------- |
| `↑` / `k`       | Move up                   |
| `↓` / `j`       | Move down                 |
| `g` / `Home`     | Jump to top               |
| `G` / `End`      | Jump to bottom            |
| `Enter`          | Connect to selected host  |
| `q` / `Esc`      | Quit                      |

## Development

```sh
make build   # build binary
make test    # run tests (100% coverage required)
make lint    # run golangci-lint
```

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## License

[MIT](LICENSE)
