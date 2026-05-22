CREATE TABLE IF NOT EXISTS daily_limits (
    user_id VARCHAR(36) PRIMARY KEY,
    daily_limit NUMERIC NOT NULL,
    currency VARCHAR(3) NOT NULL
);
