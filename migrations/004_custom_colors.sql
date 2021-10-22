-- Write your migrate up statements here

ALTER TABLE guild_settings 
    ADD COLUMN message_color VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN error_color VARCHAR(255) NOT NULL DEFAULT '';

---- create above / drop below ----

ALTER TABLE guild_settings
    DROP COLUMN message_color,
    DROP COLUMN error_color;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
