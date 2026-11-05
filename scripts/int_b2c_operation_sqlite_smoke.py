#!/usr/bin/env python3
"""Offline B2.C SQLite idempotency and crash-state proposal tests; NO runtime."""
import sqlite3
from pathlib import Path

root=Path(__file__).resolve().parents[1]
db=sqlite3.connect(':memory:',isolation_level=None)
db.execute('PRAGMA foreign_keys=ON')
db.execute('CREATE TABLE workspace(id TEXT PRIMARY KEY)')
db.execute("INSERT INTO workspace VALUES ('ws1')")
db.executescript((root/'docs/contracts/dubbridge-fenix-b2a2-principal-sqlite-proposal.up.sql').read_text())
db.execute("INSERT INTO service_principal (id,consumer,workspace_id,dubbridge_org_id,created_at_unix) VALUES ('p1','dubbridge','ws1','org1',2000000000)")
db.execute("INSERT INTO service_principal (id,consumer,workspace_id,dubbridge_org_id,created_at_unix) VALUES ('p2','dubbridge','ws1','org1',2000000000)")
db.executescript((root/'docs/contracts/dubbridge-fenix-b2c-operation-proposal.up.sql').read_text())
key='operation-00000001'
digest=b'1'*32
new=lambda p,k,h: (p,k,'ws1','integration.diagnostics','1',h,'trace-001',2000000000,2000000000)
insert="""INSERT INTO service_operation
(principal_id,operation_id,workspace_id,capability_key,schema_version,request_sha256,trace_id,created_at_unix,updated_at_unix)
VALUES (?,?,?,?,?,?,?,?,?) ON CONFLICT(principal_id,operation_id) DO NOTHING"""
read='SELECT request_sha256,status,run_id FROM service_operation WHERE principal_id=? AND operation_id=?'
assert db.execute(insert,new('p1',key,digest)).rowcount==1
assert db.execute(insert,new('p1',key,digest)).rowcount==0
assert db.execute(read,('p1',key)).fetchone()==(digest,'accepted',None)
assert db.execute(insert,new('p1',key,b'2'*32)).rowcount==0
assert db.execute(read,('p1',key)).fetchone()[0]!=b'2'*32
assert db.execute(insert,new('p2',key,digest)).rowcount==1
assert db.execute('SELECT COUNT(*) FROM service_operation').fetchone()[0]==2
rejected=0
for sql,args in [
    (insert,new('missing',key,digest)),
    (insert,new('p1','short',digest)),
    (insert,new('p1','operation-00000002',b'x')),
    ("UPDATE service_operation SET status='succeeded' WHERE principal_id=? AND operation_id=?",('p1',key)),
    ("UPDATE service_operation SET lease_owner='a' WHERE principal_id=? AND operation_id=?",('p1',key)),
]:
    try: db.execute(sql,args)
    except sqlite3.IntegrityError:rejected+=1
    else:raise AssertionError('unexpected acceptance')
assert rejected==5
# Do not auto-retry an expired lease; ambiguous accepted work is marked indeterminate
assert db.execute("UPDATE service_operation SET status='indeterminate',updated_at_unix=? WHERE principal_id=? AND operation_id=? AND status='accepted'",(2000000010,'p1',key)).rowcount==1
assert db.execute(read,('p1',key)).fetchone()[1]=='indeterminate'
assert db.execute("UPDATE service_operation SET status='running',run_id='run-real',updated_at_unix=? WHERE principal_id=? AND operation_id=? AND status='accepted'",(2000000011,'p1',key)).rowcount==0
assert db.execute('SELECT COUNT(*) FROM service_operation').fetchone()[0]==2
db.executescript((root/'docs/contracts/dubbridge-fenix-b2c-operation-proposal.down.sql').read_text())
assert db.execute("SELECT name FROM sqlite_master WHERE name='service_operation'").fetchone() is None
print('B2C_SQLITE_PROPOSAL_PASS=1 negative=5 replay=PASS scope=PASS indeterminate=PASS rollback=PASS')
