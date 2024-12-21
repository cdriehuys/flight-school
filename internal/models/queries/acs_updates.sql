-- name: UpsertACS :one
INSERT INTO acs (id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name
RETURNING *;

-- name: UpsertArea :one
INSERT INTO acs_areas (acs_id, public_id, name, "order")
VALUES ($1, $2, $3, $4)
ON CONFLICT (acs_id, public_id) DO UPDATE
SET name = EXCLUDED.name, "order" = EXCLUDED."order"
RETURNING *;

-- name: ClearUnknownAreas :execrows
DELETE FROM acs_areas
WHERE acs_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));

-- name: UpsertTask :one
INSERT INTO acs_area_tasks (area_id, public_id, name, objective, note)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (area_id, public_id) DO UPDATE
SET name = EXCLUDED.name, objective = EXCLUDED.objective, note = EXCLUDED.note
RETURNING *;

-- name: ClearUnknownTasks :execrows
DELETE FROM acs_area_tasks
WHERE area_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));

-- name: UpsertTaskReference :one
INSERT INTO task_references (task_id, document, "order")
VALUES ($1, $2, $3)
ON CONFLICT (task_id, "order") DO UPDATE
SET document = EXCLUDED.document
RETURNING *;

-- name: ClearUnknownTaskReferences :execrows
DELETE FROM task_references
WHERE task_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));

-- name: UpsertTaskElement :one
INSERT INTO acs_elements (task_id, "type", public_id, content)
VALUES ($1, $2, $3, $4)
ON CONFLICT (task_id, "type", public_id) DO UPDATE
SET content = EXCLUDED.content
RETURNING *;

-- name: ClearUnknownTaskElements :execrows
DELETE FROM acs_elements
WHERE task_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));

-- name: UpsertSubElement :one
INSERT INTO acs_subelements (element_id, "order", content)
VALUES ($1, $2, $3)
ON CONFLICT (element_id, "order") DO UPDATE
SET content = EXCLUDED.content
RETURNING *;

-- name: ClearUnknownSubElements :execrows
DELETE FROM acs_subelements
WHERE element_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));
