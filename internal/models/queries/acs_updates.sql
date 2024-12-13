-- name: UpsertTaskReference :one
INSERT INTO task_references (task_id, document, "order")
VALUES ($1, $2, $3)
ON CONFLICT (task_id, "order") DO UPDATE
SET document = EXCLUDED.document
RETURNING *;

-- name: ClearUnknownTaskReferences :execrows
DELETE FROM task_references
WHERE task_id = $1 AND NOT (id = ANY(sqlc.arg(known_ids)::int[]));
