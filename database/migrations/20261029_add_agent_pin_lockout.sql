USE go_pulsa_db;

ALTER TABLE agents
    ADD COLUMN pin_failed_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN pin_locked_until TIMESTAMP NULL DEFAULT NULL;
