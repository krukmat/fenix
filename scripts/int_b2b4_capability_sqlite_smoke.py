#!/usr/bin/env python3
"""Offline B2.B4 SQLite policy gates. Uses REAL migrations 038-040, no Go/app/route."""
import hashlib
import sqlite3
import tempfile
import threading
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MIGRATIONS = ROOT / "internal/infra/sqlite/migrations"
CHECKS = 0


def pass_check(label, condition):
    global CHECKS
    assert condition, label
    CHECKS += 1
    print("PASS", CHECKS, label)


def refused(db, label, sql, args=()):
    try:
        db.execute(sql, args)
    except sqlite3.IntegrityError:
        pass_check(label, True)
    else:
        raise AssertionError("unexpected acceptance: " + label)


def setup(filename):
    db = sqlite3.connect(filename, timeout=5, check_same_thread=False)
    db.execute("PRAGMA foreign_keys=ON")
    db.execute("PRAGMA busy_timeout=5000")
    # Isolated parent stub; 038-040 below are the exact Fenix SQL files.
    db.execute("CREATE TABLE workspace(id TEXT PRIMARY KEY)")
    db.executemany("INSERT INTO workspace(id) VALUES(?)", [("w1",), ("w2",)])
    for version, name in [
        ("038", "service_identity"), ("039", "service_operation"),
        ("040", "s2s_capability_policy"),
    ]:
        sql = (MIGRATIONS / (version + "_" + name + ".up.sql")).read_text()
        db.executescript(sql)
    db.execute("""INSERT INTO service_principal
        (id,consumer,workspace_id,dubbridge_org_id,created_at_unix)
        VALUES ('p1','dubbridge','w1','org1',2000000000)""")
    db.execute("""INSERT INTO service_principal
        (id,consumer,workspace_id,dubbridge_org_id,status,created_at_unix,revoked_at_unix)
        VALUES ('p2','dubbridge','w1','org2','revoked',2000000000,2000000001)""")
    db.commit()
    return db


SCHEMA = ('{"type":"object","properties":{"v":{"type":"integer"}},'
          '"required":["v"],"additionalProperties":false}')
INSERT = """INSERT INTO service_s2s_capability_policy(
    principal_id,workspace_id,capability_key,schema_version,handler_id,
    schema_json,schema_sha256,updated_by,created_at_unix,updated_at_unix)
    VALUES (?,?,?,?,?,?,?,?,?,?)"""
UPDATE = """UPDATE service_s2s_capability_policy
    SET enabled=?,revision=revision+1,updated_by=?,updated_at_unix=?
    WHERE principal_id=? AND schema_version=?"""


def provision(db, principal="p1", version="1", workspace="w1"):
    return db.execute(INSERT, (
        principal, workspace, "integration.diagnostics", version, "test-handler",
        SCHEMA, hashlib.sha256(SCHEMA.encode()).digest(),
        "operator", 2000000000, 2000000000,
    ))


def set_enabled(db, version, enabled, at):
    return db.execute(UPDATE, (enabled, "operator", at, "p1", version))


with tempfile.TemporaryDirectory(prefix="fenix-b2b4-q2-") as directory:
    filename = str(Path(directory) / "policy.sqlite")
    db = setup(filename)
    provision(db)
    pass_check("provision succeeds", True)
    pass_check("provision defaults to disabled revision one",
               db.execute("SELECT enabled,revision FROM service_s2s_capability_policy").fetchone() == (0, 1))
    pass_check("audit records provisioning",
               db.execute("SELECT action,revision FROM service_s2s_capability_audit").fetchall() == [("provision", 1)])
    set_enabled(db, "1", 1, 2000000001)
    pass_check("enabled policy transition", db.execute(
        "SELECT enabled,revision FROM service_s2s_capability_policy WHERE schema_version='1'"
    ).fetchone() == (1, 2))
    pass_check("audit records enabling", db.execute(
        "SELECT action FROM service_s2s_capability_audit ORDER BY id"
    ).fetchall() == [("provision",), ("enable",)])
    provision(db, version="2")
    pass_check("second version provisions disabled", db.execute(
        "SELECT enabled FROM service_s2s_capability_policy WHERE schema_version='2'"
    ).fetchone()[0] == 0)
    refused(db, "unique enabled version", UPDATE, (1, "operator", 2000000002, "p1", "2"))
    pass_check("rejected enable does not mutate row", db.execute(
        "SELECT enabled FROM service_s2s_capability_policy WHERE schema_version='2'"
    ).fetchone()[0] == 0)
    set_enabled(db, "1", 0, 2000000003)
    set_enabled(db, "2", 1, 2000000004)
    pass_check("version transition", db.execute(
        "SELECT schema_version FROM service_s2s_capability_policy WHERE enabled=1"
    ).fetchall() == [("2",)])
    for name, sql in [
        ("immutable handler", "UPDATE service_s2s_capability_policy SET handler_id='forged'"),
        ("immutable schema", "UPDATE service_s2s_capability_policy SET schema_json='{}'"),
        ("immutable digest", "UPDATE service_s2s_capability_policy SET schema_sha256=zeroblob(32)"),
        ("immutable identity", "UPDATE service_s2s_capability_policy SET principal_id='p2'"),
        ("forbid revision-only change", "UPDATE service_s2s_capability_policy SET revision=revision+1"),
        ("forbid policy deletion", "DELETE FROM service_s2s_capability_policy"),
        ("forbid audit mutation", "UPDATE service_s2s_capability_audit SET action='disable'"),
        ("forbid audit deletion", "DELETE FROM service_s2s_capability_audit"),
    ]:
        refused(db, name, sql)
    refused(db, "revoked principal cannot provision", INSERT, (
        "p2", "w1", "integration.diagnostics", "1", "test-handler", SCHEMA,
        hashlib.sha256(SCHEMA.encode()).digest(), "operator", 2000000000, 2000000000,
    ))
    refused(db, "cross-workspace principal cannot provision", INSERT, (
        "p1", "w2", "integration.diagnostics", "1", "test-handler", SCHEMA,
        hashlib.sha256(SCHEMA.encode()).digest(), "operator", 2000000000, 2000000000,
    ))
    set_enabled(db, "2", 0, 2000000005)
    db.execute("""UPDATE service_principal SET status='revoked',revoked_at_unix=2000000006
                  WHERE id='p1'""")
    refused(db, "revoked principal cannot re-enable", UPDATE, (
        1, "operator", 2000000007, "p1", "2",
    ))
    pass_check("rejected re-enable has no audit entry", db.execute(
        "SELECT action,revision FROM service_s2s_capability_audit WHERE schema_version='2'"
    ).fetchall() == [("provision", 1), ("enable", 2), ("disable", 3)])
    db.close()

    # Second database for concurrency ordering. It uses the same real migrations.
    lockfile = str(Path(directory) / "locking.sqlite")
    init = setup(lockfile)
    provision(init)
    set_enabled(init, "1", 1, 2000000001)
    init.commit()
    init.close()
    a = sqlite3.connect(lockfile, timeout=5, check_same_thread=False)
    b = sqlite3.connect(lockfile, timeout=5, check_same_thread=False)
    checked = threading.Event()
    release = threading.Event()
    disabled = threading.Event()

    def reserve():
        a.execute("BEGIN IMMEDIATE")
        assert a.execute("SELECT enabled FROM service_s2s_capability_policy").fetchone()[0] == 1
        checked.set()
        assert release.wait(3)
        a.commit()

    def disable():
        assert checked.wait(3)
        set_enabled(b, "1", 0, 2000000002)
        b.commit()
        disabled.set()

    threads = [threading.Thread(target=reserve), threading.Thread(target=disable)]
    for thread in threads:
        thread.start()
    assert checked.wait(3)
    assert not disabled.wait(0.15), "policy changed before transaction commit"
    release.set()
    for thread in threads:
        thread.join(timeout=5)
        assert not thread.is_alive()
    pass_check("BEGIN IMMEDIATE serializes disable after reservation", disabled.is_set())
    pass_check("committed disabled state visible to next reader", b.execute(
        "SELECT enabled FROM service_s2s_capability_policy"
    ).fetchone()[0] == 0)
    a.close()
    b.close()

print(f"B2B4_SQLITE_038_040_PASS={CHECKS}; sqlite_version={sqlite3.sqlite_version}")
