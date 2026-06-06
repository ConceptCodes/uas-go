-- Phase 4+ Migration: MFA, SSO, Webhooks, Admin
-- Adds tables for MFA factors, MFA challenges, identity providers,
-- user identities, webhook endpoints, and webhook deliveries.

-- MFA Factors
CREATE TABLE IF NOT EXISTS mfa_factors (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    factor_type VARCHAR(20) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    name VARCHAR(100) DEFAULT '',
    is_primary BOOLEAN DEFAULT FALSE,
    backup_codes JSON,
    last_used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_mfa_factors_user (user_id),
    INDEX idx_mfa_factors_department (department_id),
    INDEX idx_mfa_factors_user_dept (user_id, department_id)
);

-- MFA Challenges
CREATE TABLE IF NOT EXISTS mfa_challenges (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    factor_id VARCHAR(36) NULL,
    challenge_type VARCHAR(20) NOT NULL,
    state VARCHAR(20) DEFAULT 'pending',
    temp_token VARCHAR(255) UNIQUE,
    code VARCHAR(10),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_mfa_challenges_user (user_id),
    INDEX idx_mfa_challenges_department (department_id),
    INDEX idx_mfa_challenges_temp_token (temp_token),
    INDEX idx_mfa_challenges_state (state)
);

-- Identity Providers (SSO)
CREATE TABLE IF NOT EXISTS identity_providers (
    id VARCHAR(36) PRIMARY KEY,
    department_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    provider_type VARCHAR(20) NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret TEXT,
    issuer_url VARCHAR(255),
    authorization_url VARCHAR(255),
    token_url VARCHAR(255),
    user_info_url VARCHAR(255),
    jwks_uri VARCHAR(255),
    metadata_url VARCHAR(255),
    redirect_urls JSON,
    scopes JSON,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_idp_department (department_id),
    INDEX idx_idp_type (provider_type)
);

-- User Identities (linked SSO accounts)
CREATE TABLE IF NOT EXISTS user_identities (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    provider_id VARCHAR(36) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    access_token TEXT,
    refresh_token TEXT,
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_ui_user (user_id),
    INDEX idx_ui_department (department_id),
    INDEX idx_ui_provider (provider_id),
    INDEX idx_ui_provider_user (provider_id, provider_user_id),
    UNIQUE KEY uk_ui_provider_account (provider_id, provider_user_id)
);

-- Webhook Endpoints
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id VARCHAR(36) PRIMARY KEY,
    department_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    url VARCHAR(500) NOT NULL,
    secret VARCHAR(255) NOT NULL,
    events JSON NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    INDEX idx_webhook_department (department_id),
    INDEX idx_webhook_active (department_id, is_active)
);

-- Webhook Deliveries
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id VARCHAR(36) PRIMARY KEY,
    endpoint_id VARCHAR(36) NOT NULL,
    department_id VARCHAR(36) NOT NULL,
    event VARCHAR(50) NOT NULL,
    payload JSON,
    response_code INT DEFAULT 0,
    response_body TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    attempt INT DEFAULT 0,
    max_attempts INT DEFAULT 5,
    next_retry_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_wd_endpoint (endpoint_id),
    INDEX idx_wd_department (department_id),
    INDEX idx_wd_status (status),
    INDEX idx_wd_retry (status, next_retry_at)
);
