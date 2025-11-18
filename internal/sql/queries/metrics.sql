-- name: CreateMetric :exec
INSERT INTO metric (id, type, delta, value, hash)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
) ON CONFLICT (id) DO UPDATE SET delta = EXCLUDED.delta, value=EXCLUDED.value;

-- name: GetMetricByID :one
SELECT * FROM metric WHERE id = $1;

-- name: GetMetrics :many
SELECT * FROM metric;
