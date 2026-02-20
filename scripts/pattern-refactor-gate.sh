#!/usr/bin/env bash

set -euo pipefail

MODE="warn"
ROOT="."
TS_DUP_THRESHOLD="2"

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
    --ts-dup-threshold)
      TS_DUP_THRESHOLD="${2:-}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      echo "Usage: $0 [--mode warn|strict] [--root <path>] [--ts-dup-threshold <pct>]"
      exit 2
      ;;
  esac
done

if [[ "$MODE" != "warn" && "$MODE" != "strict" ]]; then
  echo "Invalid mode: $MODE (expected warn or strict)"
  exit 2
fi

cd "$ROOT"

issue_count=0
issue_log=""

add_issue() {
  issue_count=$((issue_count + 1))
  issue_log+="- $1"$'\n'
}

extract_first_number() {
  local input="$1"
  printf '%s\n' "$input" | grep -Eo '[0-9]+([.][0-9]+)?' | head -n1 || true
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

# ─── Go: structural duplication opportunities (dupl) ─────────────────────────

go_dupl_issues=0
go_dupl_available="yes"
go_dupl_sample=""

if command -v golangci-lint >/dev/null 2>&1; then
  go_dupl_out=$(golangci-lint run --enable-only=dupl --out-format line-number ./... 2>&1 || true)
  go_dupl_issues=$(printf '%s\n' "$go_dupl_out" | grep -Ec '^[^[:space:]].*\.go:[0-9]+' || true)
  go_dupl_sample=$(printf '%s\n' "$go_dupl_out" | grep -E '^[^[:space:]].*\.go:[0-9]+' | head -n 5 || true)
else
  go_dupl_available="no"
  add_issue "Go: golangci-lint no está disponible; no se pudo ejecutar dupl."
fi

if [[ "$go_dupl_available" == "yes" && "$go_dupl_issues" -gt 0 ]]; then
  add_issue "Go: dupl detectó $go_dupl_issues coincidencias estructurales (oportunidades de Extract/Template/Factory)."
fi

# ─── TypeScript: structural duplication opportunities (jscpd) ────────────────

ts_dup_available="yes"
ts_dup_percent=""
ts_dup_clones=""
ts_jscpd_report_dir=".tmp/pattern-gate/jscpd"
mkdir -p "$ts_jscpd_report_dir"

if command -v npx >/dev/null 2>&1; then
  ts_jscpd_out=$(npx --yes jscpd mobile/src bff/src \
    --min-lines 5 \
    --min-tokens 50 \
    --threshold 100 \
    --reporters console \
    --output "$ts_jscpd_report_dir" \
    --ignore "**/__tests__/**,**/*.test.ts,**/*.test.tsx,**/coverage/**,**/node_modules/**,**/dist/**,**/.expo/**" \
    2>&1 || true)

  ts_dup_percent=$(printf '%s\n' "$ts_jscpd_out" | sed -nE 's/.*\(([0-9]+([.][0-9]+)?)%\)[[:space:]]duplicated.*/\1/p' | tail -n1)
  ts_dup_clones=$(printf '%s\n' "$ts_jscpd_out" | sed -nE 's/.*Found[[:space:]]+([0-9]+)[[:space:]]+clones?.*/\1/p' | tail -n1)

  if [[ -z "$ts_dup_percent" ]]; then
    fallback_pct_line=$(printf '%s\n' "$ts_jscpd_out" | grep -Ei 'duplicat' | tail -n1 || true)
    ts_dup_percent=$(extract_first_number "$fallback_pct_line")
  fi

  if [[ -z "$ts_dup_clones" ]]; then
    fallback_clone_line=$(printf '%s\n' "$ts_jscpd_out" | grep -Ei 'found[[:space:]]+[0-9]+[[:space:]]+clone' | tail -n1 || true)
    ts_dup_clones=$(extract_first_number "$fallback_clone_line")
  fi
else
  ts_dup_available="no"
  add_issue "TypeScript: npx no está disponible; no se pudo ejecutar jscpd."
fi

if [[ "$ts_dup_available" == "yes" ]]; then
  if [[ -z "$ts_dup_percent" ]]; then
    add_issue "TypeScript: jscpd no devolvió porcentaje de duplicación parseable (revisar salida de herramienta)."
  else
    over_threshold=$(awk -v d="$ts_dup_percent" -v t="$TS_DUP_THRESHOLD" 'BEGIN { if (d+0 > t+0) print 1; else print 0 }')
    if [[ "$over_threshold" == "1" ]]; then
      add_issue "TypeScript: jscpd reporta ${ts_dup_percent}% de duplicación (umbral: ${TS_DUP_THRESHOLD}%)."
    fi
  fi
fi

# ─── report ───────────────────────────────────────────────────────────────────

echo "=== Pattern Refactor Gate (mode: $MODE) ==="
echo ""
echo "[Evidencia de refactor]"
echo "  Evidence files : $evidence_count"
echo ""
echo "[Go - dupl]"
echo "  tool available : $go_dupl_available"
echo "  issues         : $go_dupl_issues"
echo "  token threshold: 120 (configurado en .golangci.yml)"

if [[ -n "$go_dupl_sample" ]]; then
  echo "  sample:"
  printf '    %s\n' "$go_dupl_sample"
fi

echo ""
echo "[TypeScript - jscpd]"
echo "  tool available : $ts_dup_available"
echo "  duplicate %    : ${ts_dup_percent:-n/a}"
echo "  clone count    : ${ts_dup_clones:-n/a}"
echo "  gate threshold : ${TS_DUP_THRESHOLD}%"

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
