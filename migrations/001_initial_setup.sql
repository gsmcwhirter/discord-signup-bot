-- Write your migrate up statements here

CREATE TABLE guild_settings (
    guild_id CHAR(20),
    settings BYTEA,
    PRIMARY KEY(guild_id)
);

CREATE TABLE events (
    guild_id CHAR(20),
    event_name VARCHAR(255),
    event_data BYTEA,
    PRIMARY KEY(guild_id, event_name)
);

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
