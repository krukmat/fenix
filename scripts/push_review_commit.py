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
import shutil
import subprocess
import sys
import tempfile
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


def _stable_blocked_artifact_dir(today, sha):
    return os.path.join("docs", "reports", "push-review", "artifacts", f"{today}-{sha}")


def _write_text_atomic(path, content):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    fd, tmp_path = tempfile.mkstemp(prefix=".tmp-", dir=os.path.dirname(path), text=True)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(content)
        os.replace(tmp_path, path)
    except Exception:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
        raise


def _write_json_atomic(path, data):
    _write_text_atomic(path, json.dumps(data, indent=2, sort_keys=True) + "\n")


def _blocked_fallback_line(content):
    for line in content.splitlines():
        if line.startswith("| Fallback packet | "):
            return line
    return None


def _replace_fallback_line(content, fallback_path):
    old_line = _blocked_fallback_line(content)
    if not old_line:
        return content
    return content.replace(old_line, f"| Fallback packet | {fallback_path} |", 1)


def publish_blocked_artifacts(out_dir, head_sha, today):
    sha = short_sha(head_sha)
    blocked_src = os.path.join(out_dir, "blocked.json")
    if not os.path.isfile(blocked_src):
        raise RuntimeError("blocked push-review report is missing out_dir/blocked.json")

    with open(blocked_src, encoding="utf-8") as f:
        blocked = json.load(f)

    final_dir = _stable_blocked_artifact_dir(today, sha)
    artifacts_root = os.path.dirname(final_dir)
    os.makedirs(artifacts_root, exist_ok=True)
    temp_dir = tempfile.mkdtemp(prefix=f".tmp-{today}-{sha}-", dir=artifacts_root)

    try:
        blocked_rel = os.path.join(final_dir, "blocked.json")
        raw_src = ((blocked.get("forensics") or {}).get("raw_completion_path"))
        raw_rel = None
        if raw_src:
            if not os.path.isfile(raw_src):
                raise RuntimeError(f"blocked raw completion missing: {raw_src}")
            raw_rel = os.path.join(final_dir, os.path.basename(raw_src))
            shutil.copyfile(raw_src, os.path.join(temp_dir, os.path.basename(raw_src)))

        blocked["source_artifacts"] = {
            "blocked_json_path": blocked_src,
            "raw_completion_path": raw_src,
        }
        reports = dict(blocked.get("reports") or {})
        reports["blocked_json_path"] = blocked_rel
        reports["fallback_packet_path"] = blocked_rel
        blocked["reports"] = reports
        if raw_rel:
            forensics = dict(blocked.get("forensics") or {})
            forensics["raw_completion_path"] = raw_rel
            blocked["forensics"] = forensics

        temp_blocked = os.path.join(temp_dir, "blocked.json")
        _write_json_atomic(temp_blocked, blocked)

        if not os.path.isfile(temp_blocked):
            raise RuntimeError("stable blocked artifact payload was not written")

        if os.path.isdir(final_dir):
            shutil.rmtree(final_dir)
        shutil.move(temp_dir, final_dir)

        if not os.path.isfile(os.path.join(final_dir, "blocked.json")):
            raise RuntimeError("stable blocked artifact missing after publish")
        if raw_rel and not os.path.isfile(os.path.join(final_dir, os.path.basename(raw_rel))):
            raise RuntimeError("stable raw completion missing after publish")
        return blocked_rel
    except Exception:
        shutil.rmtree(temp_dir, ignore_errors=True)
        raise


def copy_report(out_dir, head_sha, today):
    sha = short_sha(head_sha)
    src = os.path.join(out_dir, "reports", f"{today}-{sha}.md")
    dst = os.path.join("docs", "reports", "push-review", f"{today}-{sha}.md")
    if os.path.isfile(src):
        with open(src, encoding="utf-8") as f:
            content = f.read()

        fallback_line = _blocked_fallback_line(content)
        if fallback_line:
            stable_blocked_path = publish_blocked_artifacts(out_dir, head_sha, today)
            content = _replace_fallback_line(content, stable_blocked_path)
            if stable_blocked_path not in content:
                raise RuntimeError("blocked summary did not retain the stable fallback packet path")
            if not os.path.isfile(stable_blocked_path):
                raise RuntimeError(f"stable fallback packet missing after publish: {stable_blocked_path}")

        _write_text_atomic(dst, content)
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


def build_commit_message(head_sha, today):
    sha = short_sha(head_sha)
    return f"[push-review] report {sha} + daily {today} entry [skip ci]"


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
        artifact_dir = os.path.join("docs", "reports", "push-review", "artifacts", f"{today}-{sha}")
        if os.path.isdir(artifact_dir):
            files_to_add.append(artifact_dir)

    subprocess.run(["git", "add"] + files_to_add, check=False)

    r = subprocess.run(["git", "diff", "--cached", "--quiet"], check=False)
    if r.returncode == 0:
        print("[push-review-commit] nothing to commit", file=sys.stderr)
        return

    subprocess.run([
        "git", "commit", "-m",
        build_commit_message(head_sha, today),
    ], check=True)
    subprocess.run(["git", "push", "origin", "HEAD:main"], check=True)
    print(f"[push-review-commit] committed and pushed", file=sys.stderr)


if __name__ == "__main__":
    main()
