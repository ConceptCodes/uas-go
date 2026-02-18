-- Migration: align_auth_schema
-- Version: 2
-- Description: Align SQL schema with current GORM models used by the API

CREATE TABLE IF NOT EXISTS department_models (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) UNIQUE,
    secret VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_department_models_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS user_models (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NULL,
    email VARCHAR(100) UNIQUE,
    password VARCHAR(255) NULL,
    phone_number VARCHAR(14) UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    encrypted_name TEXT NULL,
    encrypted_email TEXT NULL,
    encrypted_phone_number TEXT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_user_models_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Best-effort migration from legacy users table to current user_models table.
INSERT INTO user_models (
    id,
    name,
    email,
    password,
    phone_number,
    email_verified,
    created_at,
    updated_at
)
SELECT
    u.uuid,
    TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))),
    u.email,
    u.password_hash,
    u.phone_number,
    u.email_verified,
    u.created_at,
    u.updated_at
FROM users u
WHERE u.uuid IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM user_models um WHERE um.id = u.uuid
  );

CREATE TABLE IF NOT EXISTS department_roles (
    id VARCHAR(36) NOT NULL,
    role VARCHAR(10) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (id, user_id),
    INDEX idx_department_roles_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS auth_models (
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(128) NOT NULL,
    type VARCHAR(36) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    PRIMARY KEY (token, type),
    INDEX idx_auth_models_user_id (user_id),
    INDEX idx_auth_models_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS department_configs (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) UNIQUE,
    department_id VARCHAR(36) UNIQUE,
    magic_link_base_url VARCHAR(255) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_department_configs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS password_histories (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_password_histories_user_id (user_id),
    INDEX idx_password_histories_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS security_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(50) NULL,
    user_id VARCHAR(36) NULL,
    department_id VARCHAR(36) NULL,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    status VARCHAR(20) NULL,
    error_message TEXT NULL,
    request_id VARCHAR(36) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_security_events_event_type (event_type),
    INDEX idx_security_events_user_id (user_id),
    INDEX idx_security_events_department_id (department_id),
    INDEX idx_security_events_created_at (created_at),
    INDEX idx_security_events_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NULL,
    refresh_token VARCHAR(500) UNIQUE,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    expires_at DATETIME(3) NULL,
    revoked_at DATETIME(3) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    deleted_at DATETIME(3) NULL,
    INDEX idx_sessions_user_id (user_id),
    INDEX idx_sessions_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NULL,
    department_id VARCHAR(36) NULL,
    action VARCHAR(50) NULL,
    resource VARCHAR(50) NULL,
    resource_id VARCHAR(36) NULL,
    severity VARCHAR(20) NULL,
    status VARCHAR(20) NULL,
    description TEXT NULL,
    ip_address VARCHAR(45) NULL,
    user_agent TEXT NULL,
    device_id VARCHAR(36) NULL,
    session_id VARCHAR(36) NULL,
    metadata JSON NULL,
    timestamp DATETIME(3) NULL,
    created_at DATETIME(3) NULL,
    updated_at DATETIME(3) NULL,
    INDEX idx_audit_logs_user_id (user_id),
    INDEX idx_audit_logs_department_id (department_id),
    INDEX idx_audit_logs_action (action),
    INDEX idx_audit_logs_resource (resource),
    INDEX idx_audit_logs_resource_id (resource_id),
    INDEX idx_audit_logs_severity (severity),
    INDEX idx_audit_logs_status (status),
    INDEX idx_audit_logs_ip_address (ip_address),
    INDEX idx_audit_logs_device_id (device_id),
    INDEX idx_audit_logs_session_id (session_id),
    INDEX idx_audit_logs_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
