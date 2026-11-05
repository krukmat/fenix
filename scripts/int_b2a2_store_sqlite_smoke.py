#!/usr/bin/env python3
"""Real SQLite check for the proposed Fenix B2.A2 Go query text.

Offline and isolated. This script does NOT invoke Fenix's MigrateUp or any
production database, and cannot substitute for the full Go integration suite.
"""
import hashlib
import re
import sqlite3
from pathlib import Path

root = Path(__file__).resolve().parents[1]
sql_go = (root / "internal/domain/auth/service_principal_store.go").read_text()
blocks = [part.strip() for part in re.findall(r"`([^`]*)`", sql_go, flags=re.S)]

def query(prefix):
    matches = [part for part in blocks if part.startswith(prefix)]
    assert len(matches) == 1, f"expected unique query {prefix!r}, got {len(matches)}"
    return matches[0]

create_principal = query("INSERT INTO service_principal")
create_credential = query("INSERT INTO service_credential")
lookup = query("SELECT p.id")
revoke_credential = query("UPDATE service_credential")
revoke_principal = query("UPDATE service_principal")

conn = sqlite3.connect(":memory:")
conn.execute("PRAGMA foreign_keys=ON")
conn.execute("CREATE TABLE workspace(id TEXT PRIMARY KEY)")
conn.execute("INSERT INTO workspace(id) VALUES ('ws-1')")
conn.executescript((root / "docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql").read_text())
now = 2_000_000_000
conn.execute(create_principal, ("p1", "dubbridge", "ws-1", "org-1", "proj-1", now))
raw = "fenix_s2s_v1_" + "Q" * 43  # non-operational synthetic fixture; digest-only schema
hashed = hashlib.sha256(raw.encode()).digest()
assert conn.execute(create_credential, ("cred-1", hashed, now, now + 120, "p1")).rowcount == 1
row = conn.execute(lookup, (hashed, now, now)).fetchone()
assert row[:5] == ("p1", "dubbridge", "ws-1", "org-1", "proj-1")
assert row[5] == hashed
assert conn.execute(lookup, (hashed, now + 120, now + 120)).fetchone() is None
assert conn.execute(lookup, (bytes([7]) * 32, now, now)).fetchone() is None

negatives = 0
for sql, args in [
    (create_principal, ("bad-consumer", "human", "ws-1", "org", None, now)),
    (create_principal, ("bad-workspace", "dubbridge", "absent", "org", None, now)),
    (create_credential, ("bad-digest", b"short", now, now + 120, "p1")),
    (create_credential, ("bad-expiry", bytes([8]) * 32, now, now, "p1")),
    (create_credential, ("duplicate", hashed, now, now + 120, "p1")),
]:
    try:
        conn.execute(sql, args)
    except sqlite3.IntegrityError:
        negatives += 1
    else:
        raise AssertionError("SQLite accepted invalid storage")
assert negatives == 5

assert conn.execute(revoke_credential, (now + 1, "cred-1", now + 1)).rowcount == 1
assert conn.execute(lookup, (hashed, now + 2, now + 2)).fetchone() is None
assert conn.execute(revoke_credential, (now + 2, "cred-1", now + 2)).rowcount == 0

new_digest = bytes([3]) * 32
assert conn.execute(create_credential, ("cred-2", new_digest, now + 3, now + 120, "p1")).rowcount == 1
assert conn.execute(lookup, (new_digest, now + 4, now + 4)).fetchone()
assert conn.execute(revoke_principal, (now + 5, "p1", now + 5)).rowcount == 1
assert conn.execute(lookup, (new_digest, now + 6, now + 6)).fetchone() is None
assert conn.execute(create_credential, ("after-revoke", bytes([6]) * 32, now + 7, now + 120, "p1")).rowcount == 0

schema_cols = [r[1] for r in conn.execute("PRAGMA table_info(service_credential)")]
assert "token" not in schema_cols and "token_plaintext" not in schema_cols
conn.executescript((root / "docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.down.sql").read_text())
assert conn.execute("SELECT name FROM sqlite_master WHERE name='service_credential'").fetchone() is None
print(f"B2A2_REAL_SQLITE_GO_SQL_PASS=1 negative={negatives} authenticated=PASS expiry=PASS revocation=PASS rollback=PASS")
