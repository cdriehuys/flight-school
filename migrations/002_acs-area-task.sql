CREATE TABLE acs_area_tasks (
    id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    area_id INTEGER NOT NULL REFERENCES acs_areas(id)
        ON DELETE CASCADE,
    task_id CHAR(1) NOT NULL,
    "name" TEXT NOT NULL,
    objective TEXT NOT NULL,
    UNIQUE (area_id, task_id)
);

ALTER TABLE acs_area_tasks
    ADD CONSTRAINT ck_name_len CHECK (char_length("name") <= 100),
    ADD CONSTRAINT ck_objective_len CHECK (char_length("objective") <= 500);

---- create above / drop below ----

DROP TABLE acs_area_tasks;