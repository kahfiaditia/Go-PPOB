USE go_pulsa_db;

-- One-time upgrade from the legacy schema used before owner/employee accounts
-- and deposit requests. Back up the database before running this script.

ALTER TABLE admins
    ADD COLUMN email VARCHAR(100) NULL AFTER username;

UPDATE admins
SET email = CONCAT(username, '@example.invalid')
WHERE email IS NULL OR TRIM(email) = '';

ALTER TABLE admins
    MODIFY COLUMN email VARCHAR(100) NOT NULL,
    ADD UNIQUE KEY uq_admins_email (email);

UPDATE agents
SET email = CONCAT(username, '@example.invalid')
WHERE email IS NULL OR TRIM(email) = '';

ALTER TABLE agents
    MODIFY COLUMN email VARCHAR(100) NOT NULL,
    ADD COLUMN address VARCHAR(255) DEFAULT NULL AFTER phone,
    ADD COLUMN bank_name VARCHAR(100) DEFAULT NULL AFTER address,
    ADD COLUMN bank_account_number VARCHAR(50) DEFAULT NULL AFTER bank_name,
    ADD UNIQUE KEY uq_agents_email (email);

CREATE TABLE agent_users (
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

CREATE TABLE agent_product_pricing (
    agent_id BIGINT NOT NULL,
    product_id VARCHAR(20) NOT NULL,
    default_profit DECIMAL(15,2) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (agent_id, product_id),
    CONSTRAINT chk_agent_product_profit CHECK (default_profit BETWEEN 500 AND 2000),
    FOREIGN KEY (agent_id) REFERENCES agents(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB;

CREATE TABLE deposit_requests (
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

ALTER TABLE deposits
    ADD COLUMN request_id VARCHAR(36) DEFAULT NULL AFTER id;

INSERT INTO deposit_requests
    (id, agent_id, amount, reference, status, admin_id, admin_note, created_at, processed_at)
SELECT id, agent_id, amount, CONCAT('LEGACY-', id), 'APPROVED', admin_id,
       'Migrated from legacy deposit history', created_at, created_at
FROM deposits;

UPDATE deposits
SET request_id = id;

ALTER TABLE deposits
    MODIFY COLUMN request_id VARCHAR(36) NOT NULL,
    ADD UNIQUE KEY uq_deposits_request_id (request_id),
    ADD FOREIGN KEY (request_id) REFERENCES deposit_requests(id);
