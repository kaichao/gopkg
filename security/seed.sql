-- gopkg/security 默认种子数据
-- 预置 admin 和 automation 用户，仅在不存在时插入。

-- ============================================================
-- admin — 平台管理员，全权限
-- ============================================================
INSERT INTO t_user (name, display_name, status)
SELECT 'admin', 'Platform Admin', 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM t_user WHERE name = 'admin');

INSERT INTO t_role_binding (user_id, role, scope)
SELECT id, 'admin', NULL
FROM t_user WHERE name = 'admin'
AND NOT EXISTS (
    SELECT 1 FROM t_role_binding
    WHERE user_id = (SELECT id FROM t_user WHERE name = 'admin')
      AND role = 'admin'
      AND scope IS NULL
);

-- ============================================================
-- automation — agent / actuator 系统组件
-- ============================================================
INSERT INTO t_user (name, display_name, status)
SELECT 'automation', 'System Automation (agent/actuator)', 'ACTIVE'
WHERE NOT EXISTS (SELECT 1 FROM t_user WHERE name = 'automation');

INSERT INTO t_role_binding (user_id, role, scope)
SELECT id, 'automation', NULL
FROM t_user WHERE name = 'automation'
AND NOT EXISTS (
    SELECT 1 FROM t_role_binding
    WHERE user_id = (SELECT id FROM t_user WHERE name = 'automation')
      AND role = 'automation'
      AND scope IS NULL
);
