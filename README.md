# az — agent z

Jump to any AI agent session by typing a directory name — like `z` but for agent tooling.

```
az dokkai              # claude (default) session in ~/…/dokkai
az o scwroll           # opencode session in ~/…/scwroll
az g                   # browse all gemini sessions
az                     # browse all claude sessions
```

## Supported agents

| Agent | Keyword | Binary | Session source |
|-------|---------|--------|----------------|
| Claude Code | *(default)* | `claude` | `~/.claude/projects/` |
| OpenCode | `o` | `opencode` | `~/.local/share/opencode/storage/session/` |
| Gemini CLI | `g` | `gemini` | *(no session history)* |
| Codex | `c` | `codex` | *(no session history)* |
| Amp | `amp` | `amp` | *(no session history)* |

Only agents whose binary is found in `$PATH` are shown. Agents without session history still open correctly — `az` just can't pre-filter by recent use.

## TUI

`az` opens a full-terminal interactive picker:

```
  az  —  Claude Code
  dokkai                                               3/19
  ─────────────────────────────────────────────────────────
  ~/Desktop/animesh/code/dokkai                   3d ago
  ~/Desktop/animesh/code/quiz-vid/editor          1w ago
  ~/Desktop/realfast/exo-code-server              Jan 12
  ─────────────────────────────────────────────────────────
  ↑↓ navigate   enter select   esc quit   az <keyword> [query] to switch agent
```

| Key | Action |
|-----|--------|
| Type | Filter sessions live |
| `↑` / `↓` | Move cursor |
| `ctrl-p` / `ctrl-n` | Move cursor |
| `pgup` / `pgdn` | Scroll by page |
| `enter` | Open selected session |
| `esc` / `ctrl-c` | Cancel |

## Configuration

`az` auto-detects installed agents. To customise keywords or add new agents, create `~/.config/az/config.json`:

```json
{
  "default": "claude",
  "agents": [
    { "name": "claude",   "keyword": "",    "cmd": "claude",   "label": "Claude Code" },
    { "name": "opencode", "keyword": "o",   "cmd": "opencode", "label": "OpenCode" },
    { "name": "gemini",   "keyword": "g",   "cmd": "gemini",   "label": "Gemini CLI" },
    { "name": "codex",    "keyword": "c",   "cmd": "codex",    "label": "Codex" },
    { "name": "amp",      "keyword": "amp", "cmd": "amp",      "label": "Amp" }
  ]
}
```

See `az --agents` to list currently active agents and their keywords.

## Install

### 1. Build

```sh
git clone https://github.com/animeshdas/az
cd az
go build -o az .
cp az ~/.local/bin/az
```

### 2. Add the shell function

The binary prints the chosen path; the shell wrapper does `cd` + agent launch so it takes effect in your current shell.

Add to `~/.zshrc` (or `~/.bashrc`):

```sh
az() {
  local target
  target=$(command az "$@") || return $?
  echo "→ $target"
  cd "$target" && claude   # replace with your default agent if needed
}
```

For multi-agent support with the right command per agent, use:

```sh
az() {
  local target agent_cmd
  # capture both chosen path and active agent cmd from env
  target=$(command az "$@") || return $?
  echo "→ $target"
  cd "$target"
  # az prints the path; derive the command from the keyword
  case "$1" in
    o) opencode ;;
    g) gemini ;;
    c) codex ;;
    amp) amp ;;
    *) claude ;;
  esac
}
```

Then reload: `source ~/.zshrc`

## CLI reference

```
az [query]             Browse default agent sessions, pre-filtered by query
az <keyword> [query]   Browse sessions for a specific agent
az --agents            List all configured agents and their keywords
az --list              Print all default agent session paths
az <keyword> --list    Print all sessions for a specific agent
```

## Requirements

- Go 1.21+
- At least one agent CLI on your `$PATH` (`claude`, `opencode`, `gemini`, `codex`, or `amp`)
