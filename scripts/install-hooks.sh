#!/bin/bash
# Install Git hooks for agent-sdk-go
# Usage: ./scripts/install-hooks.sh

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_SRC="$REPO_ROOT/scripts/hooks"
HOOKS_DST="$REPO_ROOT/.git/hooks"

echo "Installing Git hooks..."

for hook in "$HOOKS_SRC"/*; do
  name="$(basename "$hook")"
  dst="$HOOKS_DST/$name"

  if [ -f "$dst" ] && [ ! -L "$dst" ]; then
    echo "  ⚠ $name: existing hook found, backing up to $name.bak"
    mv "$dst" "$dst.bak"
  fi

  ln -sf "$hook" "$dst"
  chmod +x "$hook"
  echo "  ✓ $name installed (symlink → scripts/hooks/$name)"
done

echo ""
echo "✓ Git hooks installed successfully."
echo "  They will run automatically on 'git commit'."
echo ""
echo "  To run manually:  .git/hooks/pre-commit"
echo "  To uninstall:     rm .git/hooks/pre-commit"
