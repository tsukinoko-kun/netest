-- name: AddHistoryEntry :exec
INSERT INTO history_entries (
    download_speed, upload_speed, latency_ms, packet_loss, jitter_ms
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetAllHistoryEntries :many
SELECT * FROM history_entries ORDER BY timestamp ASC;
