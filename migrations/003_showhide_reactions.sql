-- Write your migrate up statements here

ALTER TABLE guild_settings 
    ADD COLUMN hide_reactions_announce BOOLEAN NOT NULL DEFAULT 'f',
    ADD COLUMN hide_reactions_show BOOLEAN NOT NULL DEFAULT 'f';

ALTER TABLE events
    ADD COLUMN hide_reactions_announce BOOLEAN NOT NULL DEFAULT 'f',
    ADD COLUMN hide_reactions_show BOOLEAN NOT NULL DEFAULT 'f';

---- create above / drop below ----

ALTER TABLE guild_settings
    DROP COLUMN hide_reactions_announce,
    DROP COLUMN hide_reactions_show;

ALTER TABLE events
    DROP COLUMN hide_reactions_announce,
    DROP COLUMN hide_reactions_show;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
