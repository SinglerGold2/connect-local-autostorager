CREATE TABLE IF NOT EXISTS autostorager_imports (
    id     INTEGER PRIMARY KEY AUTOINCREMENT,
    at     TEXT NOT NULL,
    folder TEXT NOT NULL,
    source TEXT NOT NULL,
    target TEXT NOT NULL
);
