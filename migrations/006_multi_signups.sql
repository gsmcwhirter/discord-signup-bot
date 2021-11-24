-- Write your migrate up statements here

ALTER TABLE event_role_signups RENAME TO event_role_signups_bak;
DROP TRIGGER update_event_role_signups_updated_at ON event_role_signups;

CREATE TABLE event_role_signups (
    event_role_signup_id SERIAL,
    PRIMARY KEY (event_role_signup_id),
    guild_id CHAR(20),
    event_name VARCHAR(255),
    role_name VARCHAR(255),
    member_id VARCHAR(255) NOT NULL,
    KEY (guild_id, event_name, role_name),
    KEY (guild_id, member_id, event_name),
    signup_state VARCHAR(255) NOT NULL,
    signup_note TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    KEY (guild_id, event_name, created_at),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_event_role_signups_updated_at 
    BEFORE UPDATE ON event_role_signups 
    FOR EACH ROW EXECUTE PROCEDURE  update_updated_at_column();

ALTER TABLE guild_settings
    ADD COLUMN allow_multi_signups BOOLEAN NOT NULL DEFAULT 'f',
    ADD COLUMN show_notes BOOLEAN NOT NULL DEFAULT 'f';

ALTER TABLE events 
    ADD COLUMN allow_multi_signups BOOLEAN NOT NULL DEFAULT 'f',
    ADD COLUMN show_notes BOOLEAN NOT NULL DEFAULT 'f';

---- create above / drop below ----

ALTER TABLE guild_settings
    DROP COLUMN allow_multi_signups,
    DROP COLUMN show_notes;

ALTER TABLE events
    DROP COLUMN allow_multi_signups,
    DROP COLUMN show_notes;

ALTER TABLE event_role_signups_bak RENAME TO event_role_signups;

CREATE TRIGGER update_event_role_signups_updated_at 
    BEFORE UPDATE ON event_role_signups 
    FOR EACH ROW EXECUTE PROCEDURE  update_updated_at_column();

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
