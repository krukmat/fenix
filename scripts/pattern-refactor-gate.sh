#!/usr/bin/env bash

set -euo pipefail

MODE="warn"
ROOT="."

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      MODE="${2:-}"
      shift 2
      ;;
    --root)
      ROOT="${2:-}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 [--mode warn|strict] [--root <path>]"
      exit 2
      ;;
  esac
done

if [[ "$MODE" != "warn" && "$MODE" != "strict" ]]; then
  echo "Invalid mode: $MODE (expected warn or strict)"
  exit 2
fi

cd "$ROOT"

# ─── helpers ──────────────────────────────────────────────────────────────────

count_go_matches() {
  local pattern="$1"
  local raw
  raw=$(grep -R -nE --include='*.go' --exclude='*_test.go' "$pattern" ./internal ./cmd ./pkg 2>/dev/null || true)
  if [[ -z "$raw" ]]; then echo "0"; else printf '%s\n' "$raw" | wc -l | tr -d ' '; fi
}

# Count occurrences of a pattern in mobile TypeScript source.
# Excludes __tests__, node_modules, and .expo (generated code).
count_mobile_matches() {
  local pattern="$1"
  local raw
  raw=$(grep -R -nE --include='*.ts' --include='*.tsx' \
    --exclude-dir='__tests__' --exclude-dir='node_modules' --exclude-dir='.expo' \
    "$pattern" ./mobile/src ./mobile/app 2>/dev/null || true)
  if [[ -z "$raw" ]]; then echo "0"; else printf '%s\n' "$raw" | wc -l | tr -d ' '; fi
}

issue_count=0
issue_log=""

add_issue() {
  issue_count=$((issue_count + 1))
  issue_log+="- $1"$'\n'
}

# ─── evidence docs ────────────────────────────────────────────────────────────

EVIDENCE_DIR="docs/refactors"
TEMPLATE_FILE="$EVIDENCE_DIR/template.md"

evidence_count=0
if [[ -d "$EVIDENCE_DIR" ]]; then
  while IFS= read -r file; do
    evidence_count=$((evidence_count + 1))
  done < <(find "$EVIDENCE_DIR" -type f -name '*.md' ! -name 'template.md' ! -name 'README.md' | sort)
else
  add_issue "No existe $EVIDENCE_DIR (faltan evidencias de refactor con patrones)."
fi

if [[ ! -f "$TEMPLATE_FILE" ]]; then
  add_issue "No existe template de evidencia: $TEMPLATE_FILE"
fi

if [[ "$evidence_count" -eq 0 ]]; then
  add_issue "No hay evidencias de refactor en $EVIDENCE_DIR (archivos .md distintos de template.md)."
fi

required_sections=(
  "## Patrón aplicado"
  "## Problema previo"
  "## Motivación"
  "## Before"
  "## After"
  "## Riesgos y rollback"
  "## Tests"
  "## Métricas"
)

if [[ -d "$EVIDENCE_DIR" ]]; then
  while IFS= read -r file; do
    for section in "${required_sections[@]}"; do
      if ! grep -qF "$section" "$file"; then
        add_issue "$file no incluye sección obligatoria: $section"
      fi
    done
  done < <(find "$EVIDENCE_DIR" -type f -name '*.md' ! -name 'template.md' ! -name 'README.md' | sort)
fi

# ─── Go: pattern signals ──────────────────────────────────────────────────────

strategy_interfaces=$(count_go_matches 'type[[:space:]]+[A-Za-z0-9_]+Strategy[[:space:]]+interface')
factory_signals=$(count_go_matches 'type[[:space:]]+[A-Za-z0-9_]+Factory[[:space:]]+interface|func[[:space:]]+New[A-Za-z0-9_]+Factory\(')
decorator_signals=$(count_go_matches 'type[[:space:]]+[A-Za-z0-9_]+Decorator[[:space:]]+struct|func[[:space:]]+New[A-Za-z0-9_]+Decorator\(')
type_switches=$(count_go_matches 'switch[[:space:]]+.*\.\(type\)')

pattern_signals=$((strategy_interfaces + factory_signals + decorator_signals))

if [[ "$pattern_signals" -eq 0 ]]; then
  add_issue "Go: no se detectaron señales de Strategy/Factory/Decorator en el código Go (MVP)."
fi

if [[ "$type_switches" -gt 0 && "$strategy_interfaces" -eq 0 ]]; then
  add_issue "Go: hay type-switches ($type_switches) sin señales de Strategy; posible oportunidad de refactor."
fi

# ─── Mobile TypeScript: duplication signals ───────────────────────────────────

# 1. useThemeColors / useColors — inline definition per file.
#    A single shared hook in src/hooks/useThemeColors.ts should replace all copies.
#    Threshold: >= 3 local definitions means the extract has not happened yet.
mobile_theme_hook_defs=$(count_mobile_matches 'function use(ThemeColors|Colors)\(\)')
mobile_theme_hook_threshold=3
if [[ "$mobile_theme_hook_defs" -ge "$mobile_theme_hook_threshold" ]]; then
  add_issue "Mobile: useThemeColors/useColors definida $mobile_theme_hook_defs veces inline; extraer a src/hooks/useThemeColors.ts (Extract Custom Hook)."
fi

# 2. formatLatency / formatCost / formatTokens — format helpers duplicated across files.
#    A single src/utils/format.ts should own them.
#    Threshold: >= 2 definitions.
mobile_format_defs=$(count_mobile_matches 'function format(Latency|Cost|Tokens)\(')
mobile_format_threshold=2
if [[ "$mobile_format_defs" -ge "$mobile_format_threshold" ]]; then
  add_issue "Mobile: helpers de formato (formatLatency/formatCost) definidos $mobile_format_defs veces; extraer a src/utils/format.ts (Utility Extract)."
fi

# 3. getStatusColor / getStatusLabel / getPriorityColor — color/label lookup duplicated.
#    A centralized src/utils/statusColors.ts (Strategy Lookup Table) should replace all copies.
#    Threshold: >= 3 definitions.
mobile_color_defs=$(count_mobile_matches 'function get(Status|Priority)(Color|Label)\(')
mobile_color_threshold=3
if [[ "$mobile_color_defs" -ge "$mobile_color_threshold" ]]; then
  add_issue "Mobile: helpers de color/label (getStatusColor/getPriorityColor/getStatusLabel) definidos $mobile_color_defs veces; centralizar en src/utils/statusColors.ts (Strategy Lookup)."
fi

# 4. useInfiniteQuery repeated in useCRM.ts without a factory hook.
#    If there are >= 4 useInfiniteQuery calls and no createInfiniteListHook factory,
#    the Factory Method pattern has not been applied.
mobile_infinite_query_count=$(count_mobile_matches 'useInfiniteQuery\(')
mobile_factory_hook=$(count_mobile_matches 'function create(InfiniteList|List)Hook')
if [[ "$mobile_infinite_query_count" -ge 4 && "$mobile_factory_hook" -eq 0 ]]; then
  add_issue "Mobile: useInfiniteQuery repetido $mobile_infinite_query_count veces sin factory hook; crear createInfiniteListHook() en useCRM.ts (Factory Method)."
fi

# ─── report ───────────────────────────────────────────────────────────────────

echo "=== Pattern Refactor Gate (mode: $MODE) ==="
echo ""
echo "[Go]"
echo "  Evidence files : $evidence_count"
echo "  strategy       : $strategy_interfaces"
echo "  factory        : $factory_signals"
echo "  decorator      : $decorator_signals"
echo "  type switches  : $type_switches"
echo ""
echo "[Mobile TypeScript]"
echo "  useThemeColors/useColors defs : $mobile_theme_hook_defs  (threshold >= $mobile_theme_hook_threshold)"
echo "  format helper defs            : $mobile_format_defs  (threshold >= $mobile_format_threshold)"
echo "  color/label helper defs       : $mobile_color_defs  (threshold >= $mobile_color_threshold)"
echo "  useInfiniteQuery calls        : $mobile_infinite_query_count  (factory hooks: $mobile_factory_hook)"

if [[ "$issue_count" -eq 0 ]]; then
  echo ""
  echo "PASS: pattern/refactor gate sin hallazgos."
  exit 0
fi

echo ""
echo "Hallazgos ($issue_count):"
printf "%s" "$issue_log"

if [[ "$MODE" == "warn" ]]; then
  echo ""
  echo "WARN mode: no bloquea CI."
  exit 0
fi

echo ""
echo "STRICT mode: gate falló."
exit 1
