-- Create the history_entries table for storing network test results
CREATE TABLE IF NOT EXISTS history_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    download_speed REAL NOT NULL,
    upload_speed REAL NOT NULL,
    latency_ms INTEGER NOT NULL,
    packet_loss REAL NOT NULL,
    jitter_ms INTEGER NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create index on timestamp for efficient querying
CREATE INDEX IF NOT EXISTS idx_history_timestamp ON history_entries(timestamp);
