-- Demo seed — Governed Support Copilot Demo (F9.A6)
-- Idempotent: INSERT OR IGNORE — safe to run multiple times.
-- Execute: sqlite3 fenixcrm.db < internal/infra/sqlite/seed/demo_support.sql
--
-- Fixed UUIDs (v4) so the runbook can reference them explicitly.
-- All timestamps are ISO 8601 UTC.

-- ============================================================
-- WORKSPACE
-- ============================================================
INSERT OR IGNORE INTO workspace (id, name, slug, settings, created_at, updated_at)
VALUES (
    'f9d00001-0000-4000-a000-000000000001',
    'FenixCRM Demo',
    'fenix-demo',
    '{"feature_flags":{"copilot":true,"agents":true,"approvals":true}}',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

-- ============================================================
-- ROLES
-- ============================================================
INSERT OR IGNORE INTO role (id, workspace_id, name, description, permissions, created_at, updated_at)
VALUES (
    'f9d00002-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'support_agent',
    'Support agent — can view and work cases, trigger support agent',
    '{"cases":["read","write"],"agents":["trigger"],"copilot":["query"]}',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

INSERT OR IGNORE INTO role (id, workspace_id, name, description, permissions, created_at, updated_at)
VALUES (
    'f9d00002-0000-4000-a000-000000000002',
    'f9d00001-0000-4000-a000-000000000001',
    'manager',
    'Manager — can approve sensitive actions',
    '{"cases":["read","write","approve"],"agents":["trigger","approve"],"approvals":["read","decide"]}',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

-- ============================================================
-- USERS
-- password_hash below = bcrypt of "demo-password-2026"
-- ============================================================
INSERT OR IGNORE INTO user_account (
    id, workspace_id, email, password_hash, display_name, status, created_at, updated_at
)
VALUES (
    'f9d00003-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'operator@fenix-demo.io',
    '$2a$10$demohashdemohashdemohasOPERATORxxxxxxxxxxxxxxxxxxx',
    'Alex Operator',
    'active',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

INSERT OR IGNORE INTO user_account (
    id, workspace_id, email, password_hash, display_name, status, created_at, updated_at
)
VALUES (
    'f9d00003-0000-4000-a000-000000000002',
    'f9d00001-0000-4000-a000-000000000002',
    'approver@fenix-demo.io',
    '$2a$10$demohashdemohashdemohasAPPROVERxxxxxxxxxxxxxxxxxxx',
    'Morgan Approver',
    'active',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

-- Assign roles
INSERT OR IGNORE INTO user_role (id, user_id, role_id, created_at)
VALUES (
    'f9d00004-0000-4000-a000-000000000001',
    'f9d00003-0000-4000-a000-000000000001',
    'f9d00002-0000-4000-a000-000000000001',
    '2026-05-01T09:00:00Z'
);

INSERT OR IGNORE INTO user_role (id, user_id, role_id, created_at)
VALUES (
    'f9d00004-0000-4000-a000-000000000002',
    'f9d00003-0000-4000-a000-000000000002',
    'f9d00002-0000-4000-a000-000000000002',
    '2026-05-01T09:00:00Z'
);

-- ============================================================
-- ACCOUNT
-- ============================================================
INSERT OR IGNORE INTO account (
    id, workspace_id, name, domain, industry, size_segment,
    owner_id, metadata, created_at, updated_at
)
VALUES (
    'f9d00005-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'Acme Enterprise',
    'acme-enterprise.io',
    'technology',
    'enterprise',
    'f9d00003-0000-4000-a000-000000000001',
    '{"tier":"gold","arr":250000,"region":"EMEA"}',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

-- ============================================================
-- CONTACT
-- ============================================================
INSERT OR IGNORE INTO contact (
    id, workspace_id, account_id, first_name, last_name, email, phone,
    title, status, owner_id, created_at, updated_at
)
VALUES (
    'f9d00006-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'f9d00005-0000-4000-a000-000000000001',
    'Dana',
    'Chen',
    'dana.chen@acme-enterprise.io',
    '+1-555-0100',
    'Head of Engineering',
    'active',
    'f9d00003-0000-4000-a000-000000000001',
    '2026-05-01T09:00:00Z',
    '2026-05-01T09:00:00Z'
);

-- ============================================================
-- CASE (support ticket)
-- ============================================================
INSERT OR IGNORE INTO case_ticket (
    id, workspace_id, account_id, contact_id, owner_id,
    subject, description, priority, status, channel,
    metadata, created_at, updated_at
)
VALUES (
    'f9d00007-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'f9d00005-0000-4000-a000-000000000001',
    'f9d00006-0000-4000-a000-000000000001',
    'f9d00003-0000-4000-a000-000000000001',
    'Login screen broken after update',
    'After the latest platform update, enterprise users report the login screen crashes on load. Affects ~200 users in EMEA. Reproducible on Chrome 124+ and Safari 17.',
    'high',
    'open',
    'email',
    '{"tags":["enterprise","login","regression"],"sla_tier":"gold"}',
    '2026-05-01T10:30:00Z',
    '2026-05-01T10:30:00Z'
);

-- ============================================================
-- KNOWLEDGE ITEM (support KB article)
-- ============================================================
INSERT OR IGNORE INTO knowledge_item (
    id, workspace_id, source_type, title, raw_content, normalized_content,
    entity_type, entity_id, metadata, created_at, updated_at
)
VALUES (
    'f9d00008-0000-4000-a000-000000000001',
    'f9d00001-0000-4000-a000-000000000001',
    'kb_article',
    'Known login issue — cache invalidation after platform update',
    'After platform updates that modify session token format, browser-cached login pages may fail to load the new JS bundle. Resolution: instruct users to clear browser cache (Ctrl+Shift+R) or open in incognito. For enterprise accounts on gold SLA, escalate immediately if more than 50 users are affected. Patch ETA: within 24h of report.',
    'after platform updates that modify session token format browser cached login pages may fail to load the new js bundle resolution instruct users to clear browser cache or open in incognito for enterprise accounts on gold sla escalate immediately if more than 50 users are affected patch eta within 24h of report',
    'case',
    'f9d00007-0000-4000-a000-000000000001',
    '{"tags":["login","cache","enterprise","workaround"],"confidence":"high","author":"support-team"}',
    '2026-04-28T14:00:00Z',
    '2026-04-28T14:00:00Z'
);
