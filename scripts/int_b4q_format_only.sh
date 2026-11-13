#!/usr/bin/env bash
# B4.Q5 formatting-only remediation. Run ONLY in clean, isolated worktrees
# at explicitly pinned SHAs. No agent, network, CI, commit, or push required.
set -euo pipefail

mode="${1:---check}"
if [[ "$mode" != "--check" && "$mode" != "--apply" ]]; then
  echo "Usage: $0 [--check|--apply]" >&2
  exit 2
fi
for var in FENIX_REPO DUBBRIDGE_REPO FENIX_SHA DUBBRIDGE_SHA; do
  if [[ -z "${!var:-}" ]]; then
    echo "Missing required environment variable: $var" >&2
    exit 2
  fi
done
for binary in git gofmt cargo rustfmt; do
  command -v "$binary" >/dev/null || { echo "Missing toolchain binary: $binary" >&2; exit 2; }
done

verify_isolated_worktree() {
  local root="$1" expected="$2" current
  [[ -d "$root" ]] || { echo "Missing worktree: $root" >&2; exit 2; }
  current="$(git -C "$root" rev-parse HEAD)"
  if git -C "$root" symbolic-ref -q HEAD >/dev/null; then
    echo "Refusing active branch checkout at $root; use a clean detached worktree" >&2
    exit 2
  fi
  if [[ "$current" != "$expected" ]]; then
    echo "SHA mismatch at $root: expected $expected, found $current" >&2
    exit 2
  fi
  if [[ -n "$(git -C "$root" status --porcelain)" ]]; then
    echo "Nonclean worktree at $root; refusing to touch user files" >&2
    exit 2
  fi
  echo "SOURCE_OK $root $current"
}
verify_isolated_worktree "$FENIX_REPO" "$FENIX_SHA"
verify_isolated_worktree "$DUBBRIDGE_REPO" "$DUBBRIDGE_SHA"

# All files introduced for this S2S integration, not unrelated repository Go.
# gofmt -l chooses the true subset requiring a mechanical rewrite.
mapfile -t go_files < <(
  git -C "$FENIX_REPO" ls-files '*.go' |
    grep -E '^internal/(api/s2s_|domain/(auth/service_|integration/(s2s_|service_operation_))|infra/sqlite/(s2s_|service_operation_|service_identity_))' || true
)
if (( ${#go_files[@]} == 0 )); then
  echo "No scoped S2S Go files found" >&2
  exit 2
fi
mapfile -t pending_go < <(
  cd "$FENIX_REPO" && gofmt -l "${go_files[@]}"
)
echo "GOFMT_CANDIDATES=${#pending_go[@]}"
printf '  %s\n' "${pending_go[@]}"

rust_paths=(
  crates/connectors/src/fenix_s2s.rs
  apps/api/src/routes/workspace.rs
  apps/gateway/src/proxy.rs
)
for path in "${rust_paths[@]}"; do
  [[ -f "$DUBBRIDGE_REPO/$path" ]] || { echo "Missing Rust file $path" >&2; exit 2; }
done

if [[ "$mode" == "--check" ]]; then
  if (cd "$DUBBRIDGE_REPO" && cargo fmt --all -- --check); then
    rust_code=0
  else
    rust_code=1
  fi
  if (( ${#pending_go[@]} || rust_code )); then
    echo "FORMAT_CHECK=FAIL (mechanical changes required)"
    exit 1
  fi
  echo "FORMAT_CHECK=PASS"
  exit 0
fi

echo "Formatting in isolated worktrees only; no commit or push."
if (( ${#pending_go[@]} )); then
  (cd "$FENIX_REPO" && gofmt -w "${pending_go[@]}")
fi
(cd "$DUBBRIDGE_REPO" && rustfmt --edition 2024 "${rust_paths[@]}")
echo "FENIX_CHANGED:"
git -C "$FENIX_REPO" diff --name-only
echo "DUBBRIDGE_CHANGED:"
git -C "$DUBBRIDGE_REPO" diff --name-only
git -C "$FENIX_REPO" diff --check
git -C "$DUBBRIDGE_REPO" diff --check

# Refuse any changed path outside the exact formatting allowlists.
while IFS= read -r changed; do
  [[ -n "$changed" ]] || continue
  allowed=0
  for file in "${go_files[@]}"; do
    if [[ "$file" == "$changed" ]]; then allowed=1; break; fi
  done
  if (( !allowed )); then
    echo "Unexpected Fenix changed path: $changed" >&2
    exit 1
  fi
done < <(git -C "$FENIX_REPO" diff --name-only)
while IFS= read -r changed; do
  [[ -n "$changed" ]] || continue
  allowed=0
  for file in "${rust_paths[@]}"; do
    if [[ "$file" == "$changed" ]]; then allowed=1; break; fi
  done
  if (( !allowed )); then
    echo "Unexpected DubBridge changed path: $changed" >&2
    exit 1
  fi
done < <(git -C "$DUBBRIDGE_REPO" diff --name-only)
if ! git -C "$FENIX_REPO" diff -w --quiet -- '*.go'; then
  echo "GO_SEMANTIC_CHANGE_DETECTED: inspect diff; do not publish" >&2
  exit 1
fi
if ! git -C "$DUBBRIDGE_REPO" diff -w --quiet -- '*.rs'; then
  echo "RUST_NON_WHITESPACE_CHANGE_DETECTED: inspect diff; do not publish" >&2
  exit 1
fi

if [[ -n "$(cd "$FENIX_REPO" && gofmt -l "${go_files[@]}")" ]]; then
  echo "GOFMT_NOT_CLEAN after apply" >&2
  exit 1
fi
if (cd "$DUBBRIDGE_REPO" && cargo fmt --all -- --check); then
  echo "FORMAT_PATCH_READY: review and publish with the pinned SHA pair"
else
  echo "RUSTFMT_WORKSPACE_STILL_FAILS: inspect remaining paths; do not waive" >&2
  exit 1
fi
