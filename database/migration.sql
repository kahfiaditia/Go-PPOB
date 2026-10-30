CREATE DATABASE IF NOT EXISTS go_pulsa_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE go_pulsa_db;

-- ============================================
-- AGENTS
-- ============================================
CREATE TABLE IF NOT EXISTS agents (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    pin_hash VARCHAR(255) DEFAULT NULL,
    pin_failed_attempts INT NOT NULL DEFAULT 0,
    pin_locked_until TIMESTAMP NULL DEFAULT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20) DEFAULT NULL,
    address VARCHAR(255) DEFAULT NULL,
    bank_name VARCHAR(100) DEFAULT NULL,
    bank_account_number VARCHAR(50) DEFAULT NULL,
    balance DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    callback_url VARCHAR(255) DEFAULT NULL,
    callback_secret VARCHAR(64) DEFAULT NULL,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT chk_agents_username_min_length CHECK (CHAR_LENGTH(username) >= 7),
    UNIQUE KEY uq_agents_email (email)
) ENGINE=InnoDB;

-- ============================================
-- AGENT USERS (pegawai; owner uses the agent account)
-- ============================================
CREATE TABLE IF NOT EXISTS agent_users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    agent_id BIGINT NOT NULL,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT chk_agent_users_username_min_length CHECK (CHAR_LENGTH(username) >= 7),
    UNIQUE KEY uq_agent_users_username (agent_id, username),
    FOREIGN KEY (agent_id) REFERENCES agents(id)
) ENGINE=InnoDB;

-- ============================================
-- ADMINS (Distributor side)
-- ============================================
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role ENUM('SUPERADMIN','ADMIN') DEFAULT 'ADMIN',
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_admins_username_min_length CHECK (CHAR_LENGTH(username) >= 7),
    UNIQUE KEY uq_admins_email (email)
) ENGINE=InnoDB;

-- ============================================
-- PRODUCTS
-- ============================================
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(20) PRIMARY KEY,
    operator VARCHAR(20) NOT NULL,
    nominal DECIMAL(15,2) NOT NULL,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(15,2) NOT NULL,
    harga_jual DECIMAL(15,2) NOT NULL,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_operator (operator)
) ENGINE=InnoDB;

-- ============================================
-- AGENT PRODUCT PRICING (keuntungan default agen)
-- ============================================
CREATE TABLE IF NOT EXISTS agent_product_pricing (
    agent_id BIGINT NOT NULL,
    product_id VARCHAR(20) NOT NULL,
    default_profit DECIMAL(15,2) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, product_id),
    CONSTRAINT chk_agent_product_profit CHECK (default_profit BETWEEN 500 AND 2000),
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB;

-- ============================================
-- TRANSACTIONS (Top-up pulsa ke customer)
-- ============================================
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    agent_id BIGINT NOT NULL,
    product_id VARCHAR(20) NOT NULL,
    customer_no VARCHAR(20) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    sell_price DECIMAL(15,2) NOT NULL,
    markup DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    status ENUM('PENDING','SUCCESS','FAILED','REFUNDED') DEFAULT 'PENDING',
    failure_reason VARCHAR(255) DEFAULT NULL,
    callback_status ENUM('NOT_SENT','SENT','FAILED') DEFAULT 'NOT_SENT',
    callback_attempts INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (product_id) REFERENCES products(id),
    INDEX idx_agent_created (agent_id, created_at),
    INDEX idx_status (status),
    INDEX idx_created (created_at)
) ENGINE=InnoDB;

-- ============================================
-- DEPOSIT REQUESTS (approval workflow)
-- ============================================
CREATE TABLE IF NOT EXISTS deposit_requests (
    id VARCHAR(36) PRIMARY KEY,
    agent_id BIGINT NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    reference VARCHAR(100) NOT NULL COMMENT 'No. transfer / bukti',
    status ENUM('PENDING','APPROVED','REJECTED') NOT NULL DEFAULT 'PENDING',
    admin_id BIGINT DEFAULT NULL,
    admin_note VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL DEFAULT NULL,
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (admin_id) REFERENCES admins(id),
    INDEX idx_deposit_requests_status_created (status, created_at),
    INDEX idx_deposit_requests_agent_created (agent_id, created_at),
    UNIQUE KEY uq_deposit_request_agent_reference (agent_id, reference)
) ENGINE=InnoDB;

-- ============================================
-- DEPOSITS (approved top-up saldo agent)
-- ============================================
CREATE TABLE IF NOT EXISTS deposits (
    id VARCHAR(36) PRIMARY KEY,
    request_id VARCHAR(36) NOT NULL UNIQUE,
    agent_id BIGINT NOT NULL,
    admin_id BIGINT NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    balance_before DECIMAL(15,2) NOT NULL,
    balance_after DECIMAL(15,2) NOT NULL,
    reference VARCHAR(100) DEFAULT NULL COMMENT 'No. transfer / bukti',
    note VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (admin_id) REFERENCES admins(id),
    FOREIGN KEY (request_id) REFERENCES deposit_requests(id),
    INDEX idx_agent_created (agent_id, created_at)
) ENGINE=InnoDB;

-- ============================================
-- CALLBACK LOGS (untuk audit & retry)
-- ============================================
CREATE TABLE IF NOT EXISTS callback_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tx_id VARCHAR(36) NOT NULL,
    agent_id BIGINT NOT NULL,
    url VARCHAR(255) NOT NULL,
    http_status INT DEFAULT 0,
    response_body TEXT,
    attempt INT NOT NULL,
    success TINYINT(1) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_tx (tx_id)
) ENGINE=InnoDB;

-- ============================================
-- BALANCE LEDGER (audit trail semua perubahan saldo)
-- ============================================
CREATE TABLE IF NOT EXISTS balance_ledger (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    agent_id BIGINT NOT NULL,
    ref_type ENUM('TRANSACTION','DEPOSIT','REFUND','ADJUSTMENT') NOT NULL,
    ref_id VARCHAR(36) NOT NULL,
    amount DECIMAL(15,2) NOT NULL COMMENT 'Negatif = keluar, Positif = masuk',
    balance_before DECIMAL(15,2) NOT NULL,
    balance_after DECIMAL(15,2) NOT NULL,
    note VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    INDEX idx_agent (agent_id, created_at)
) ENGINE=InnoDB;