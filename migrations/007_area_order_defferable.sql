-- Area order must be deferrable to allow for updates where an area is inserted
-- or removed from the middle of the list. Areas are special because they have
-- two unique constraints. The other constraints do not need to be deferrable
-- because we can upsert in those cases.

ALTER TABLE acs_areas
    DROP CONSTRAINT acs_areas_acs_id_order_key,
    ADD CONSTRAINT acs_areas_acs_id_order_key UNIQUE(acs_id, "order") DEFERRABLE INITIALLY DEFERRED;

---- create above / drop below ----

ALTER TABLE acs_areas
    DROP CONSTRAINT acs_areas_acs_id_order_key,
    ADD CONSTRAINT acs_areas_acs_id_order_key UNIQUE(acs_id, "order");
