#!/usr/bin/env python3
"""
Copies the push-review markdown report to docs/reports/push-review/,
creates docs/daily/<date>.md if it doesn't exist, and appends a row
to the push-review table in section 3 of that daily.

Usage:
    python3 scripts/push_review_commit.py <out_dir> <head_sha> <run_id>
"""
import json
import os
import subprocess
import sys
from datetime import datetime, timezone


def short_sha(sha):
    return sha[:7] if sha else "unknown"


def today_utc():
    return datetime.now(timezone.utc).strftime("%Y-%m-%d")


def read_aggregate(out_dir):
    path = os.path.join(out_dir, "aggregate.json")
    if not os.path.isfile(path):
        return None
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def git_branch():
    r = subprocess.run(
        ["git", "rev-parse", "--abbrev-ref", "HEAD"],
        capture_output=True, text=True, check=False,
    )
    return r.stdout.strip() or "main"


def copy_report(out_dir, head_sha, today):
    sha = short_sha(head_sha)
    src = os.path.join(out_dir, "reports", f"{today}-{sha}.md")
    dst = os.path.join("docs", "reports", "push-review", f"{today}-{sha}.md")
    if os.path.isfile(src):
        os.makedirs(os.path.dirname(dst), exist_ok=True)
        with open(src, encoding="utf-8") as f:
            content = f.read()
        with open(dst, "w", encoding="utf-8") as f:
            f.write(content)
        return dst
    return None


def create_daily(daily_path, today):
    branch = git_branch()
    content = f"""\
# Daily — {today}

**Branch:** {branch} · **Sync:** synced · **Gates:** `fmt:❓ docs:❓`
**Foco del día:** (sembrado automáticamente por push-review)

---

## 1. Roadmap pulse

- **Fase activa:** por determinar
- **Desbloquea al cerrar:** —
- **Gates de fundación en riesgo:** ninguno
- **X-items que se movieron:** —

---

## 2. Pipelines GH rotos

| Workflow | Último fallo | Estado | Acción |
|---|---|---|---|
| — | — | limpio | — |

---

## 3. Push-review post-pipeline

| Run / SHA | Conclusión pipeline | Estado push-review | RRI / routing | Acción |
|---|---|---|---|---|

---

## 4. Ayer → Hoy

| Estado | Task | Banda RRI | Nota |
|---|---|---|---|

---

## 5. Issues ledger

| Hora | Sev | Tipo | Descripción | Estado | Acción |
|---|---|---|---|---|---|

---

## 6. Optimizaciones y mejoras

| ID | Tipo | Propuesta | Impacto | Esfuerzo | → Task? |
|---|---|---|---|---|---|

---

## 7. Decisiones pendientes (HITL gate)

- [ ] (ninguna al abrir)

---

## 8. Cierre del día ✓

- [ ] `git status` limpio — sin trabajo declarado "done" sin commitear
- [ ] Roadmap ↔ ledgers ↔ git consistentes (drift-check emite 0 🔴)
- [ ] Pipelines GH rotos revisados; si existe alguno, quedó con owner o task
- [ ] Push-review más reciente revisado; findings no-pure-Low y patches `in_review` registrados o referenciados
- [ ] Gates verdes: fmt, lint, test, check, deny, secrets, cov, docs — o BLOCKER abierto
- [ ] X-items tocados hoy reflejados en roadmap
- [ ] Daily de mañana sembrado con lo `[~]` que queda
"""
    os.makedirs(os.path.dirname(daily_path), exist_ok=True)
    with open(daily_path, "w", encoding="utf-8") as f:
        f.write(content)
    print(f"[push-review-commit] created {daily_path}", file=sys.stderr)


def append_daily_row(daily_path, row):
    with open(daily_path, encoding="utf-8") as f:
        content = f.read()

    header = "| Run / SHA | Conclusión pipeline | Estado push-review | RRI / routing | Acción |"
    sep = "|---|---|---|---|---|"

    if header not in content:
        print(f"[push-review-commit] section-3 table not found in {daily_path}, skipping row", file=sys.stderr)
        return

    idx_header = content.find(header)
    idx_sep = content.find(sep, idx_header)
    if idx_sep == -1:
        print(f"[push-review-commit] separator not found, skipping row", file=sys.stderr)
        return

    # Find end of existing table rows
    after_sep = content[idx_sep + len(sep):]
    insert_offset = idx_sep + len(sep)
    for line in after_sep.split("\n"):
        if line.startswith("|"):
            insert_offset += len(line) + 1
        else:
            break

    content = content[:insert_offset] + "\n" + row + content[insert_offset:]
    with open(daily_path, "w", encoding="utf-8") as f:
        f.write(content)
    print(f"[push-review-commit] row appended to {daily_path}", file=sys.stderr)


def main():
    if len(sys.argv) != 4:
        print(f"Usage: {sys.argv[0]} <out_dir> <head_sha> <run_id>", file=sys.stderr)
        sys.exit(1)

    out_dir, head_sha, run_id = sys.argv[1], sys.argv[2], sys.argv[3]
    today = today_utc()
    sha = short_sha(head_sha)
    daily_path = os.path.join("docs", "daily", f"{today}.md")

    # 1. Copy report
    report_dst = copy_report(out_dir, head_sha, today)
    if report_dst:
        report_link = f"[{today}-{sha}.md]({report_dst})"
        print(f"[push-review-commit] report copied to {report_dst}", file=sys.stderr)
    else:
        report_link = "(no report generated)"
        print(f"[push-review-commit] no report found in {out_dir}/reports/", file=sys.stderr)

    # 2. Read aggregate
    agg = read_aggregate(out_dir)
    if agg:
        status = agg.get("status", "?")
        audit = agg.get("audit") or {}
        quorum = audit.get("quorum", "?")
        passes_ok = audit.get("passes_succeeded", "?")
        passes_run = audit.get("passes_run", "?")
        passes = f"{passes_ok}/{passes_run}"
        routings = list({c.get("routing", "?") for c in agg.get("candidates", [])})
        routing = ", ".join(routings) if routings else "none"
        ci_conclusion = (agg.get("pipeline") or {}).get("conclusion", "?")
    else:
        status = "blocked"; quorum = "?"; passes = "?/?"; routing = "?"; ci_conclusion = "?"

    row = f"| `{run_id} / {sha}` | {ci_conclusion} | {status} ({passes} passes, quorum {quorum}) | {routing} | {report_link} |"

    # 3. Create daily if needed
    if not os.path.isfile(daily_path):
        create_daily(daily_path, today)

    # 4. Append row
    append_daily_row(daily_path, row)

    # 5. Stage files
    files_to_add = [daily_path]
    if report_dst:
        files_to_add.append(report_dst)

    subprocess.run(["git", "add"] + files_to_add, check=False)

    r = subprocess.run(["git", "diff", "--cached", "--quiet"], check=False)
    if r.returncode == 0:
        print("[push-review-commit] nothing to commit", file=sys.stderr)
        return

    subprocess.run([
        "git", "commit", "-m",
        f"chore(push-review): report {sha} + daily {today} entry [skip ci]",
    ], check=True)
    subprocess.run(["git", "push", "origin", "HEAD:main"], check=True)
    print(f"[push-review-commit] committed and pushed", file=sys.stderr)


if __name__ == "__main__":
    main()
