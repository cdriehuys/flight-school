ALTER TABLE acs_area_tasks
    ADD COLUMN note TEXT NOT NULL DEFAULT '';

---- create above / drop below ----

ALTER TABLE acs_area_tasks
    DROP COLUMN note;
