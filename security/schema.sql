-- gopkg/security 统一安全表 DDL
-- 所有应用（exastore、go-scalebox）共用此 schema。
-- 应用自身的业务表（t_project、t_cluster、t_app 等）由各应用独立维护。
--
-- 使用方式：
--   psql -f schema.sql
--   psql -f seed.sql   （可选：预置 admin/automation 用户）

-- ============================================================
-- 1. t_user — 用户
-- ============================================================
CREATE TABLE IF NOT EXISTS t_user (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    email         TEXT,
    display_name  TEXT,
    password_hash TEXT,
    status        TEXT NOT NULL DEFAULT 'ACTIVE',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_status CHECK (status IN ('ACTIVE', 'DISABLED'))
);

-- ============================================================
-- 2. t_api_key — API 密钥
--    scope_type + scope_id 限定密钥的作用范围（可选）：
--      NULL / NULL   — 不限定，等同于用户的全权限
--      'project'     — 限定到指定项目（exastore）
--      'app'         — 限定到指定应用（go-scalebox）
-- ============================================================
CREATE TABLE IF NOT EXISTS t_api_key (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES t_user(id) ON DELETE CASCADE,
    key_hash      TEXT NOT NULL,
    key_prefix    TEXT,
    name          TEXT NOT NULL,
    scope_type    TEXT,
    scope_id      TEXT,
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_api_key_hash ON t_api_key(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_key_user_id ON t_api_key(user_id);
CREATE INDEX IF NOT EXISTS idx_api_key_scope ON t_api_key(scope_type, scope_id);

-- ============================================================
-- 3. t_role_binding — 角色绑定
--    scope: JSONB 数组，表达多维作用域约束。
--      null                                                     — 全局（平台管理员）
--      [{"type":"cluster","id":"cluster-a"}]                    — 限定集群
--      [{"type":"app","id":"42"}]                               — 限定应用
--      [{"type":"app","id":"42"},{"type":"cluster","id":"cluster-a"}] — 交集
--      [{"type":"project","id":"proj-abc"}]                     — 限定项目
-- ============================================================
CREATE TABLE IF NOT EXISTS t_role_binding (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER NOT NULL REFERENCES t_user(id) ON DELETE CASCADE,
    role          TEXT NOT NULL,
    scope         JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uniq_role_binding UNIQUE NULLS NOT DISTINCT (user_id, role, scope)
);

CREATE INDEX IF NOT EXISTS idx_role_binding_user_id ON t_role_binding(user_id);
CREATE INDEX IF NOT EXISTS idx_role_binding_scope ON t_role_binding USING GIN (scope);

-- ============================================================
-- 4. t_token_blacklist — JWT 令牌黑名单
--    gopkg pgTokenBlacklist 通过 jti + expires_at 查询。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_token_blacklist (
    id            SERIAL PRIMARY KEY,
    jti           TEXT NOT NULL,
    user_id       INTEGER NOT NULL REFERENCES t_user(id) ON DELETE CASCADE,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_blacklist_jti ON t_token_blacklist(jti);
CREATE INDEX IF NOT EXISTS idx_blacklist_expires ON t_token_blacklist(expires_at);

-- ============================================================
-- 5. t_audit_log — 审计日志
--    gopkg pgAuditStore 写入 user_id / action / resource / detail / created_at。
--    user_id 无 FK：允许记录已删除用户或系统匿名操作。
-- ============================================================
CREATE TABLE IF NOT EXISTS t_audit_log (
    id            SERIAL PRIMARY KEY,
    user_id       INTEGER,
    action        TEXT NOT NULL,
    resource      TEXT,
    detail        JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON t_audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON t_audit_log(created_at);
