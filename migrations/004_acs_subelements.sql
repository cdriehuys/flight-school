CREATE TABLE acs_subelements (
    id INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    element_id INTEGER NOT NULL REFERENCES acs_elements(id)
        ON DELETE CASCADE,
    "order" INTEGER NOT NULL,
    content TEXT NOT NULL,
    UNIQUE (element_id, "order")
);

ALTER TABLE acs_subelements
    ADD CONSTRAINT ck_content_len CHECK (char_length("content") <= 500);

---- create above / drop below ----

DROP TABLE acs_subelements;
