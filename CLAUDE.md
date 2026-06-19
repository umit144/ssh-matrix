# ssh-matrix

A terminal UI app for browsing and connecting to SSH hosts from `~/.ssh/config`.

## Tech Stack
- **Language:** Go 1.26+
- **TUI Framework:** Bubble Tea (charmbracelet/bubbletea)
- **Styling:** Lip Gloss (charmbracelet/lipgloss)
- **Build:** `go build -o ssh-matrix .`
- **Run:** `go run .`
- **Test:** `go test ./...`

## Design Language
- Fullscreen alt-screen terminal app, content centered
- Minimal color palette — monochrome base with subtle accent (#82AAFF)
- Clean, slick, fancy — should look polished
- Keyboard-driven: arrows/jk to navigate, enter to connect, / to filter, q to quit

## Project Structure
```
main.go                    # entry point
internal/tui/app.go        # Bubble Tea model, update, view (with scrolling)
internal/tui/styles.go     # Lip Gloss style definitions
internal/ssh/types.go      # SSH host config types
internal/ssh/parser.go     # SSH config parser (Include, Match, quotes, dedup)
internal/ssh/parser_test.go
```

## Status
- [x] Project scaffold + Go module
- [x] Fullscreen TUI with scrolling, empty state
- [x] SSH config parser (Include, Match, multi-pattern Host, quotes, inline comments, dedup)
- [x] SSH connect on enter (with error display)
- [x] Production audit (no mock data, rune-safe truncation, scroll, panic guards)
- [x] CI/CD (lint, 100% test coverage gate, cross-build macOS/Linux, GitHub Releases)
- [ ] Filter/search (/ key)
- [ ] Config details pane
