-- Write your migrate up statements here

ALTER TABLE guild_settings
    ADD COLUMN open_admin_access BOOLEAN NOT NULL DEFAULT 'f';

---- create above / drop below ----

ALTER TABLE guild_settings
    DROP COLUMN open_admin_access;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
