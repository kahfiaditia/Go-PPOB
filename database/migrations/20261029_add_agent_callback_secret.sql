USE go_pulsa_db;

ALTER TABLE agents
    ADD COLUMN callback_secret VARCHAR(64) DEFAULT NULL;
