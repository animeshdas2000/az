# cz

Jump to any Claude session by typing a directory name — like `z` but for `claude`.

```
cz dokkai          # opens the most recent Claude session in ~/…/dokkai
cz ex              # multiple matches → pick from a list
cz                 # no query → pick from all sessions
```

## How it works

Claude stores project sessions under `~/.claude/projects/`. `cz` reads those directories, resolves each one back to its real filesystem path (using `sessions-index.json` when available, greedy slug decoding otherwise), sorts by most-recently-used, and opens `claude` in the chosen directory.

When multiple directories match your query, `cz` shows an interactive picker — `fzf` if it's installed, a numbered list otherwise.

## Install

### 1. Build

```sh
git clone https://github.com/animeshdas/cz
cd cz
go build -o cz .
cp cz ~/.local/bin/cz   # or any directory on your $PATH
```

### 2. Add the shell function

The binary outputs a path; the shell function handles the `cd` + `claude` so it affects your current shell session.

Add to your `~/.zshrc` (or `~/.bashrc`):

```sh
cz() {
  local target
  target=$(command cz "$@") || return $?
  echo "→ $target"
  cd "$target" && claude
}
```

Then reload: `source ~/.zshrc`

## Conflict resolution

If multiple directories share the same name (e.g. `exo-help` at two different paths), all matches are shown sorted by recency:

```
Multiple matches:
  1) /Users/you/realfast/exo-help
  2) /Users/you/realfast/exo-help/packages/frontend
Choose:
```

With `fzf` installed the picker is interactive.

## Requirements

- Go 1.21+
- `claude` CLI on your `$PATH`
- `fzf` (optional, for a nicer picker)
