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
    email VARCHAR(100) DEFAULT NULL,
    phone VARCHAR(20) DEFAULT NULL,
    balance DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    callback_url VARCHAR(255) DEFAULT NULL,
    callback_secret VARCHAR(64) DEFAULT NULL,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- ============================================
-- ADMINS (Distributor side)
-- ============================================
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    role ENUM('SUPERADMIN','ADMIN') DEFAULT 'ADMIN',
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- ============================================
-- PRODUCTS
-- ============================================
CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(20) PRIMARY KEY,
    operator VARCHAR(20) NOT NULL,
    nominal DECIMAL(15,2) NOT NULL,
    name VARCHAR(100) NOT NULL,
    base_price DECIMAL(15,2) NOT NULL,
    default_markup DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_operator (operator)
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
-- DEPOSITS (Top-up saldo agent oleh admin)
-- ============================================
CREATE TABLE IF NOT EXISTS deposits (
    id VARCHAR(36) PRIMARY KEY,
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