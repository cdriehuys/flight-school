CREATE TABLE acs_areas (
    id TEXT PRIMARY KEY,
    "name" TEXT NOT NULL
);

ALTER TABLE acs_areas
    ADD CONSTRAINT ck_id_len CHECK (char_length(id) <= 25),
    ADD CONSTRAINT ck_name_len CHECK (char_length("name") <= 100);

CREATE INDEX idx_id ON acs_areas (id text_pattern_ops);

INSERT INTO acs_areas(id, "name")
VALUES
    ('PA.I', 'Preflight Preparation'),
    ('PA.II', 'Preflight Procedures'),
    ('PA.III', 'Airport and Seaport Base Operations'),
    ('PA.IV', 'Takeoffs, Landings, and Go-Arounds'),
    ('PA.V', 'Performance Maneuvers and Ground Reference Maneuvers'),
    ('PA.VI', 'Navigation'),
    ('PA.VII', 'Slow Flight and Stalls'),
    ('PA.VIII', 'Basic Instrument Maneuvers'),
    ('PA.IX', 'Emergency Operations'),
    ('PA.X', 'Multiengine Operations'),
    ('PA.XI', 'Night Operations'),
    ('PA.XII', 'Postflight Procedures');

---- create above / drop below ----

DROP TABLE acs_areas;
