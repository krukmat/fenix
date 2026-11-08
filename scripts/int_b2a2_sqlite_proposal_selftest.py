#!/usr/bin/env python3
"""Offline B2.A2 proposal check. No Fenix app/migrations/credentials are changed."""
import sqlite3
from pathlib import Path

BASE = Path(__file__).resolve().parents[1] / "docs" / "contracts"
STEM = "dubbridge-fenix-b2a2-principal-sqlite-proposal"
db = sqlite3.connect(":memory:")
db.execute("PRAGMA foreign_keys=ON")
db.execute("CREATE TABLE workspace(id TEXT PRIMARY KEY)")
db.execute("INSERT INTO workspace VALUES ('workspace-test')")
db.executescript((BASE / (STEM + ".up.sql")).read_text())
now = 2000000000
db.execute(
    "INSERT INTO service_principal(id,consumer,workspace_id,dubbridge_org_id,created_at_unix) "
    "VALUES (?,?,?,?,?)", ("p1", "dubbridge", "workspace-test", "org-test", now)
)
digest = bytes([8]) * 32
db.execute(
    "INSERT INTO service_credential(id,principal_id,version,token_sha256,issued_at_unix,expires_at_unix) "
    "VALUES (?,?,?,?,?,?)", ("c1", "p1", 1, digest, now, now + 3600)
)

checks = 0
def must_reject(sql, args):
    global checks
    try:
        db.execute(sql, args)
    except sqlite3.IntegrityError:
        checks += 1
        return
    raise AssertionError("invalid data was accepted")

pinsert = ("INSERT INTO service_principal"
           "(id,consumer,workspace_id,dubbridge_org_id,created_at_unix) "
           "VALUES (?,?,?,?,?)")
cinsert = ("INSERT INTO service_credential"
           "(id,principal_id,version,token_sha256,issued_at_unix,expires_at_unix) "
           "VALUES (?,?,?,?,?,?)")
must_reject(pinsert, ("bad-consumer", "human", "workspace-test", "org-test", now))
must_reject(pinsert, ("bad-workspace", "dubbridge", "missing", "org-test", now))
must_reject(cinsert, ("duplicate", "p1", 2, digest, now, now + 3600))
must_reject(cinsert, ("short", "p1", 2, b"x" * 12, now, now + 3600))
must_reject(cinsert, ("expiry", "p1", 2, b"y" * 32, now, now))
must_reject(cinsert, ("second-active", "p1", 2, b"z" * 32, now, now + 3600))
assert checks == 6

db.execute("UPDATE service_credential SET revoked_at_unix=? WHERE id='c1'", (now + 1,))
db.execute("UPDATE service_principal SET status='revoked',revoked_at_unix=? WHERE id='p1'", (now + 2,))
assert db.execute(
    "SELECT count(*) FROM service_credential c JOIN service_principal p "
    "ON c.principal_id=p.id WHERE c.token_sha256=? AND "
    "c.revoked_at_unix IS NULL AND p.status='active'",
    (digest,)
).fetchone()[0] == 0
assert "token" not in [c[1] for c in db.execute("PRAGMA table_info(service_credential)")]
db.executescript((BASE / (STEM + ".down.sql")).read_text())
assert db.execute("SELECT name FROM sqlite_master WHERE name='service_credential'").fetchone() is None
print("B2A2_SQLITE_PROPOSAL=PASS negatives=6 revocation=PASS rollback=PASS")
