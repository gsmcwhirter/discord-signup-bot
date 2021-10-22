-- Write your migrate up statements here

ALTER TABLE events
    ADD COLUMN event_time TEXT NOT NULL DEFAULT '';

---- create above / drop below ----

ALTER TABLE events
    DROP COLUMN event_time;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
