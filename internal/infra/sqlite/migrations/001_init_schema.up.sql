-- Migration 001: Foundation schema — Tenant & Auth
-- Task 1.2.6: workspace, user_account, role, user_role tables
-- These are the core multi-tenancy and authentication entities (ERD Section 2)

-- schema_migrations tracking table (created by MigrateUp before running any migration)
-- Listed here for documentation; created programmatically by the migrator.

-- ========================
-- WORKSPACE (tenant root)
-- ========================
-- Every resource in FenixCRM belongs to a workspace.
-- workspace_id is the primary multi-tenancy isolation key.
CREATE TABLE IF NOT EXISTS workspace (
    id          TEXT    NOT NULL PRIMARY KEY,   -- UUID v7
    name        TEXT    NOT NULL,
    slug        TEXT    NOT NULL UNIQUE,        -- URL-safe identifier, e.g. "acme-corp"
    settings    TEXT,                           -- JSON: feature flags, limits, branding
    created_at  TEXT    NOT NULL,               -- ISO 8601 UTC
    updated_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_slug ON workspace (slug);

-- ========================
-- USER ACCOUNT
-- ========================
-- Represents a human user within a workspace.
-- email is UNIQUE globally (one account per email address).
-- password_hash is nullable: NULL when using external OIDC (Keycloak, P1).
CREATE TABLE IF NOT EXISTS user_account (
    id               TEXT    NOT NULL PRIMARY KEY,  -- UUID v7
    workspace_id     TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    external_idp_id  TEXT    UNIQUE,                -- External OIDC subject (nullable)
    email            TEXT    NOT NULL UNIQUE,        -- Login identifier
    password_hash    TEXT,                           -- bcrypt hash (NULL = OIDC-only)
    display_name     TEXT    NOT NULL,
    avatar_url       TEXT,
    status           TEXT    NOT NULL DEFAULT 'active'  -- active | suspended | deactivated
                         CHECK (status IN ('active', 'suspended', 'deactivated')),
    preferences      TEXT,                           -- JSON: UI preferences
    created_at       TEXT    NOT NULL,
    updated_at       TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_user_account_workspace   ON user_account (workspace_id);
CREATE INDEX IF NOT EXISTS idx_user_account_email       ON user_account (email);
CREATE INDEX IF NOT EXISTS idx_user_account_status      ON user_account (workspace_id, status);

-- ========================
-- ROLE
-- ========================
-- RBAC role definitions scoped to a workspace.
-- permissions is a JSON object: { "accounts": ["read","write"], "cases": ["read"] }
CREATE TABLE IF NOT EXISTS role (
    id           TEXT    NOT NULL PRIMARY KEY,   -- UUID v7
    workspace_id TEXT    NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,               -- UNIQUE per workspace (enforced below)
    description  TEXT,
    permissions  TEXT    NOT NULL DEFAULT '{}',  -- JSON: object/field/action grants
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL,
    UNIQUE (workspace_id, name)
);

CREATE INDEX IF NOT EXISTS idx_role_workspace ON role (workspace_id);

-- ========================
-- USER ROLE (many-to-many)
-- ========================
-- Assigns roles to users within a workspace.
-- A user can have multiple roles; a role can be assigned to multiple users.
CREATE TABLE IF NOT EXISTS user_role (
    id         TEXT    NOT NULL PRIMARY KEY,   -- UUID v7
    user_id    TEXT    NOT NULL REFERENCES user_account(id) ON DELETE CASCADE,
    role_id    TEXT    NOT NULL REFERENCES role(id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL,
    UNIQUE (user_id, role_id)                  -- no duplicate assignments
);

CREATE INDEX IF NOT EXISTS idx_user_role_user ON user_role (user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role ON user_role (role_id);
