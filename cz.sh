# cz - Claude session jumper
# Source this file in your ~/.zshrc or ~/.bashrc:
#   source /path/to/cz.sh
#
# Usage:
#   cz [query]   Jump to the most recent Claude session matching <query>.
#   cz           Show all sessions to pick from.

cz() {
  local target
  target=$(command cz "$@") || return $?
  echo "→ $target"
  cd "$target" && claude
}
