-- name: GetAreaByPublicID :one
SELECT *
FROM acs_areas
WHERE acs_id = $1 AND public_id = $2;

-- name: ListAreasByACS :many
WITH areas AS (
    SELECT * FROM acs_areas
    WHERE acs_areas.acs_id = $1
), task_count AS (
    SELECT a.id AS area_id, COUNT(t.id) as tasks
    FROM acs_area_tasks t
        LEFT JOIN areas a ON t.area_id = a.id
    GROUP BY a.id
), votes AS (
    SELECT t.area_id AS area_id, SUM(c.vote) AS votes
    FROM element_confidence c
        LEFT JOIN acs_elements e ON c.element_id = e.id
        LEFT JOIN acs_area_tasks t ON e.task_id = t.id
    WHERE t.area_id = ANY(SELECT id FROM areas)
    GROUP BY t.area_id
), max_votes AS (
    SELECT t.area_id AS area_id, COUNT(e.id) * 3 AS max_votes
    FROM acs_elements e
        LEFT JOIN acs_area_tasks t ON e.task_id = t.id
    WHERE t.area_id = ANY(SELECT id FROM areas)
    GROUP BY t.area_id
)
SELECT
    sqlc.embed(a),
    COALESCE((SELECT tasks FROM task_count WHERE area_id = a.id), 0)::int AS task_count,
    COALESCE((SELECT votes FROM votes WHERE area_id = a.id), 0)::int AS votes,
    COALESCE((SELECT max_votes FROM max_votes WHERE area_id = a.id), 0)::int as max_votes
FROM acs_areas a
WHERE a.id = ANY(SELECT id FROM areas)
ORDER BY a."order" ASC;

-- name: ListTasksByArea :many
WITH task_element_counts AS (
    SELECT e.task_id AS task_id, e.type AS "type", COUNT(e.id) AS "count"
    FROM acs_elements e
        LEFT JOIN acs_area_tasks t ON e.task_id = t.id
    WHERE t.area_id = $1
    GROUP BY e.task_id, e.type
), max_votes AS (
    SELECT COALESCE(COUNT(e.id) * 3, 0) AS max_votes, e.task_id AS task_id
    FROM acs_elements e
        LEFT JOIN acs_area_tasks t ON e.task_id = t.id
    WHERE t.area_id = $1
    GROUP BY e.task_id
), votes AS (
    SELECT e.task_id AS task_id, COALESCE(SUM(c.vote), 0) AS votes
    FROM acs_elements e
        LEFT JOIN element_confidence c ON e.id = c.element_id
        LEFT JOIN acs_area_tasks t ON e.task_id = t.id
    WHERE t.area_id = $1
    GROUP BY e.task_id
)
SELECT
    sqlc.embed(t),
    sqlc.embed(a),
    (a.acs_id || '.' || a.public_id || '.' || t.public_id)::text AS full_public_id,
    COALESCE((SELECT votes FROM votes WHERE task_id = t.id), 0)::int AS votes,
    COALESCE((SELECT max_votes FROM max_votes WHERE task_id = t.id), 0)::int AS max_votes,
    COALESCE((SELECT "count" FROM task_element_counts WHERE task_id = t.id AND "type" = 'K'), 0)::int AS knowledge_element_count,
    COALESCE((SELECT "count" FROM task_element_counts WHERE task_id = t.id AND "type" = 'R'), 0)::int AS risk_element_count,
    COALESCE((SELECT "count" FROM task_element_counts WHERE task_id = t.id AND "type" = 'S'), 0)::int AS skill_element_count
FROM acs_area_tasks t
    LEFT JOIN acs_areas a ON t.area_id = a.id
WHERE t.area_id = $1
ORDER BY t.public_id ASC;

-- name: GetTaskByPublicID :one
WITH tasks AS (
    SELECT t.id AS id
    FROM acs_area_tasks t
        LEFT JOIN acs_areas a ON t.area_id = a.id
    WHERE a.acs_id = sqlc.arg(acs)::text AND a.public_id = sqlc.arg(area_id)::text AND t.public_id = sqlc.arg(task_id)::text
), max_votes AS (
    SELECT COALESCE(COUNT(id) * 3, 0) AS max_votes, task_id
    FROM acs_elements
    WHERE task_id = ANY(SELECT id FROM tasks)
    GROUP BY task_id
), votes AS (
    SELECT e.task_id AS task_id, COALESCE(SUM(c.vote), 0) AS votes
    FROM acs_elements e
        LEFT JOIN element_confidence c ON e.id = c.element_id
    WHERE e.task_id = ANY(SELECT id FROM tasks)
    GROUP BY e.task_id
)
SELECT
    sqlc.embed(t),
    sqlc.embed(a),
    COALESCE((SELECT votes FROM votes), 0)::int AS votes,
    COALESCE((SELECT max_votes FROM max_votes ), 0)::int AS max_votes
FROM acs_area_tasks t
    LEFT JOIN acs_areas a ON t.area_id = a.id
WHERE t.id = ANY(SELECT id FROM tasks);

-- name: GetTaskByElementID :one
WITH tasks AS (
    SELECT t.id AS id
    FROM acs_area_tasks t
        JOIN acs_elements e ON t.id = e.task_id
    WHERE e.id = $1
), max_votes AS (
    SELECT COALESCE(COUNT(id) * 3, 0) AS max_votes, task_id
    FROM acs_elements
    WHERE task_id = ANY(SELECT id FROM tasks)
    GROUP BY task_id
), votes AS (
    SELECT e.task_id AS task_id, COALESCE(SUM(c.vote), 0) AS votes
    FROM acs_elements e
        LEFT JOIN element_confidence c ON e.id = c.element_id
    WHERE e.task_id = ANY(SELECT id FROM tasks)
    GROUP BY e.task_id
)
SELECT
    sqlc.embed(t),
    sqlc.embed(a),
    COALESCE((SELECT votes FROM votes), 0)::int AS votes,
    COALESCE((SELECT max_votes FROM max_votes), 0)::int AS max_votes
FROM acs_area_tasks t
    LEFT JOIN acs_areas a ON t.area_id = a.id
WHERE t.id = ANY(SELECT id FROM tasks);

-- name: GetTaskConfidenceByTaskID :one
WITH task_elements AS (
    SELECT id FROM acs_elements WHERE task_id = $1
), max_votes AS (
    SELECT COALESCE(COUNT(*) * 3, 0) AS max_votes FROM task_elements
)
SELECT
    COALESCE(SUM(c.vote), 0)::int AS votes,
    (SELECT max_votes FROM max_votes)::int AS possible
FROM element_confidence c
WHERE c.element_id IN (SELECT id FROM task_elements);

-- name: ListElementsByTaskIDs :many
SELECT *
FROM acs_elements
WHERE task_id = ANY ($1::int[])
ORDER BY "type", public_id ASC;

-- name: SetElementConfidence :exec
INSERT INTO element_confidence (element_id, vote)
VALUES ($1, $2)
ON CONFLICT (element_id) DO UPDATE
SET vote = EXCLUDED.vote;

-- name: ListSubElementsByElementIDs :many
SELECT *
FROM acs_subelements
WHERE element_id = ANY ($1::int[])
ORDER BY public_id ASC;
