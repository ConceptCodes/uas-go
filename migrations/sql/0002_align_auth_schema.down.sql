-- Migration: align_auth_schema (rollback)
-- Version: 2

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS security_events;
DROP TABLE IF EXISTS password_histories;
DROP TABLE IF EXISTS department_configs;
DROP TABLE IF EXISTS auth_models;
DROP TABLE IF EXISTS department_roles;
DROP TABLE IF EXISTS user_models;
DROP TABLE IF EXISTS department_models;
