-- Migration: baseline_v1
-- Version: 3
-- Created: 2026-06-06
-- Description: Add department_id columns, NOT NULL constraints, and composite indexes

-- Phase 2.1: Add department_id to tables that were created without it
ALTER TABLE auth_models
    ADD COLUMN department_id VARCHAR(36) NULL AFTER type,
    ADD INDEX idx_auth_models_department_id (department_id);

ALTER TABLE password_histories
    ADD COLUMN department_id VARCHAR(36) NULL AFTER user_id,
    ADD INDEX idx_password_histories_department_id (department_id);

-- Phase 1.5: Add missing NOT NULL constraints
ALTER TABLE user_models
    MODIFY email VARCHAR(100) NOT NULL,
    MODIFY password VARCHAR(255) NOT NULL;

ALTER TABLE sessions
    MODIFY department_id VARCHAR(36) NOT NULL,
    MODIFY refresh_token VARCHAR(500) NOT NULL,
    MODIFY expires_at DATETIME(3) NOT NULL;

ALTER TABLE department_models
    MODIFY name VARCHAR(100) NOT NULL;

ALTER TABLE security_events
    MODIFY event_type VARCHAR(50) NOT NULL;

-- Phase 1.6: Add unique indexes for tenant-isolated uniqueness
-- Users within a department must have unique email (enforced via department_roles join)
-- The composite PK on department_roles(id, user_id) already enforces this.
-- Add unique constraint on department_configs(department_id) — already has UNIQUE index.

-- Phase 1.7: Add composite indexes for query performance
CREATE INDEX idx_sessions_department_user ON sessions (department_id, user_id);
CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE INDEX idx_security_events_department_created ON security_events (department_id, created_at);

CREATE INDEX idx_auth_models_user_type ON auth_models (user_id, type);

CREATE INDEX idx_password_histories_user_department ON password_histories (user_id, department_id);

CREATE INDEX idx_department_roles_user ON department_roles (user_id);

CREATE INDEX idx_audit_logs_department_timestamp ON audit_logs (department_id, timestamp);
