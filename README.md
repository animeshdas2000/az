# az — agent z

Jump to any Claude session by typing a directory name — like `z` but for `claude`.

```
az dokkai          # pre-filter to sessions matching "dokkai"
az ex              # multiple matches → TUI picker filtered to "ex"
az                 # no query → browse all sessions
```

## TUI

`az` opens a full-terminal interactive picker:

```
  az  —  agent z
  dokkai                                               3/19
  ─────────────────────────────────────────────────────────
  ~/Desktop/animesh/code/dokkai                   3d ago
  ~/Desktop/animesh/code/quiz-vid/editor          1w ago
  ~/Desktop/realfast/exo-code-server              Jan 12
  ─────────────────────────────────────────────────────────
  ↑↓ navigate   enter select   esc quit
```

| Key | Action |
|-----|--------|
| Type | Filter sessions live |
| `↑` / `↓` | Move cursor |
| `ctrl-p` / `ctrl-n` | Move cursor (vim-style) |
| `pgup` / `pgdn` | Scroll by page |
| `enter` | Open selected session |
| `esc` / `ctrl-c` | Cancel |

Sessions are sorted by most-recently-used. The dim prefix shows the parent path; the bold name is the project directory; the right column shows relative age.

## How it works

Claude stores project sessions under `~/.claude/projects/`. `az` reads those directories, resolves each one back to its real filesystem path (using `sessions-index.json` when available, greedy slug decoding otherwise for older projects), sorts by most-recently-used, and launches the TUI.

The binary prints the chosen path to stdout; the shell wrapper does the `cd` + `claude` so it takes effect in your current shell.

## Install

### 1. Build

```sh
git clone https://github.com/animeshdas/az
cd az
go build -o az .
cp az ~/.local/bin/az   # or any directory on your $PATH
```

### 2. Add the shell function

Add to your `~/.zshrc` (or `~/.bashrc`):

```sh
az() {
  local target
  target=$(command az "$@") || return $?
  echo "→ $target"
  cd "$target" && claude
}
```

Then reload: `source ~/.zshrc`

## Requirements

- Go 1.21+
- `claude` CLI on your `$PATH`
