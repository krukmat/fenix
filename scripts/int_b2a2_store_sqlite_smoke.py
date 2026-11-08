#!/usr/bin/env python3
"""Run the Fenix service credential store's real modernc SQLite smoke tests."""

import os
import subprocess
from pathlib import Path

root = Path(__file__).resolve().parents[1]
env = os.environ.copy()
env.setdefault("GOTOOLCHAIN", "go1.25.13")
subprocess.run(
    ["go", "test", "-count=1", "-run", "TestServiceCredential(SQLiteLifecycle|ConcurrentRotationAndRollback|SanitizedFailures)$", "./internal/domain/auth"],
    cwd=root,
    env=env,
    check=True,
)
print("SERVICE_CREDENTIAL_SQLITE_SMOKE=PASS")
