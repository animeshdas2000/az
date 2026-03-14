# az - agent z: Claude session jumper
# Source this file in your ~/.zshrc or ~/.bashrc:
#   source /path/to/az.sh
#
# Usage:
#   az [query]   Jump to the most recent Claude session matching <query>.
#   az           Show all sessions to pick from.

az() {
  local target
  target=$(command az "$@") || return $?
  echo "→ $target"
  cd "$target" && claude
}
