-- Migration: baseline_v1 (Rollback)
-- Version: 3
-- Description: Revert department_id columns, NOT NULL constraints, and composite indexes

-- Phase 1.7: Drop composite indexes
DROP INDEX IF EXISTS idx_audit_logs_department_timestamp ON audit_logs;
DROP INDEX IF EXISTS idx_department_roles_user ON department_roles;
DROP INDEX IF EXISTS idx_password_histories_user_department ON password_histories;
DROP INDEX IF EXISTS idx_auth_models_user_type ON auth_models;
DROP INDEX IF EXISTS idx_security_events_department_created ON security_events;
DROP INDEX IF EXISTS idx_sessions_expires_at ON sessions;
DROP INDEX IF EXISTS idx_sessions_department_user ON sessions;

-- Phase 1.5: Revert NOT NULL constraints
ALTER TABLE security_events
    MODIFY event_type VARCHAR(50) NULL;

ALTER TABLE department_models
    MODIFY name VARCHAR(100) NULL;

ALTER TABLE sessions
    MODIFY expires_at DATETIME(3) NULL,
    MODIFY refresh_token VARCHAR(500) NULL,
    MODIFY department_id VARCHAR(36) NULL;

ALTER TABLE user_models
    MODIFY password VARCHAR(255) NULL,
    MODIFY email VARCHAR(100) NULL;

-- Phase 2.1: Remove department_id columns
ALTER TABLE password_histories
    DROP INDEX idx_password_histories_department_id,
    DROP COLUMN IF EXISTS department_id;

ALTER TABLE auth_models
    DROP INDEX idx_auth_models_department_id,
    DROP COLUMN IF EXISTS department_id;
