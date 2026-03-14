# az — agent z

Jump to any Claude session by typing a directory name — like `z` but for `claude`.

```
az dokkai          # opens the most recent Claude session in ~/…/dokkai
az ex              # multiple matches → interactive TUI picker
az                 # no query → browse all sessions
```

## How it works

Claude stores project sessions under `~/.claude/projects/`. `az` reads those directories, resolves each one back to its real filesystem path (using `sessions-index.json` when available, greedy slug decoding otherwise), sorts by most-recently-used, and opens an interactive TUI to pick from.

Type to filter, `↑↓` to navigate, `enter` to open, `esc` to cancel.

## Install

### 1. Build

```sh
git clone https://github.com/animeshdas/az
cd az
go build -o az .
cp az ~/.local/bin/az   # or any directory on your $PATH
```

### 2. Add the shell function

The binary outputs a path; the shell function handles the `cd` + `claude` so it affects your current shell session.

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
