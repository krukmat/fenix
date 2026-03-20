#!/bin/bash
# switch-agent.sh — declara qué agente está activo para el tracking de commits.
#
# Uso:
#   bash scripts/switch-agent.sh claude   → claude-sonnet-4-6
#   bash scripts/switch-agent.sh gpt      → gpt-codex
#   bash scripts/switch-agent.sh human    → human
#   bash scripts/switch-agent.sh          → muestra el agente actual

CURRENT="$(git config fenix.ai-agent 2>/dev/null || echo "unknown")"

if [ -z "$1" ]; then
  echo "Agente actual: $CURRENT"
  echo ""
  echo "Uso: bash scripts/switch-agent.sh [claude|gpt|human|<modelo-custom>]"
  exit 0
fi

case "$1" in
  claude) AGENT="claude-sonnet-4-6" ;;
  gpt)    AGENT="gpt-codex" ;;
  human)  AGENT="human" ;;
  *)      AGENT="$1" ;;
esac

git config fenix.ai-agent "$AGENT"
echo "Agente seteado: $AGENT"
