-- Write your migrate up statements here

ALTER TABLE guild_settings 
    ADD COLUMN command_indicator VARCHAR(10) NOT NULL DEFAULT '',
    ADD COLUMN announce_channel VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN signup_channel VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN admin_channel VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN announce_to VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN show_after_signup BOOLEAN NOT NULL DEFAULT 'f',
    ADD COLUMN show_after_withdraw BOOLEAN NOT NULL DEFAULT 'f';

ALTER TABLE events
    ADD COLUMN nice_name VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN event_state VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN announce_channel VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN signup_channel VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN announce_to VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN description TEXT NOT NULL DEFAULT '',
    ADD COLUMN role_sort_order TEXT NOT NULL DEFAULT '';

CREATE TABLE guild_admin_roles (
    guild_id CHAR(20),
    admin_role VARCHAR(255),
    PRIMARY KEY(guild_id, admin_role)
);

CREATE TABLE event_roles (
    guild_id CHAR(20),
    event_name VARCHAR(255),
    role_name VARCHAR(255),
    role_count INT NOT NULL DEFAULT 0,
    role_emoji VARCHAR(255) NOT NULL DEFAULT '',
    PRIMARY KEY (guild_id, event_name, role_name)
);

CREATE TABLE event_role_signups (
    guild_id CHAR(20),
    event_name VARCHAR(255),
    role_name VARCHAR(255),
    member_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (guild_id, event_name, role_name, member_id),
    signup_state VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_updated_at_column()   
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;   
END;
$$ language 'plpgsql';

CREATE TRIGGER update_event_role_signups_updated_at 
    BEFORE UPDATE ON event_role_signups 
    FOR EACH ROW EXECUTE PROCEDURE  update_updated_at_column();

---- create above / drop below ----

DROP TRIGGER update_event_role_signups_updated_at;

DROP FUNCTION update_updated_at_column;

DROP TABLE event_role_signups;

DROP TABLE event_roles;

DROP TABLE guild_admin_roles;

ALTER TABLE events
    DROP COLUMN nice_name,
    DROP COLUMN event_state,
    DROP COLUMN announce_channel,
    DROP COLUMN signup_channel,
    DROP COLUMN announce_to,
    DROP COLUMN description,
    DROP COLUMN role_sort_order;

ALTER TABLE guild_settings 
    DROP COLUMN command_indicator,
    DROP COLUMN announce_channel,
    DROP COLUMN signup_channel,
    DROP COLUMN admin_channel,
    DROP COLUMN announce_to,
    DROP COLUMN admin_role,
    DROP COLUMN show_after_signup,
    DROP COLUMN show_after_withdraw;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
