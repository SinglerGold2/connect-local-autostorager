CREATE TABLE IF NOT EXISTS autostorager_imports (
    id     BIGSERIAL PRIMARY KEY,
    at     TEXT NOT NULL,
    folder TEXT NOT NULL,
    source TEXT NOT NULL,
    target TEXT NOT NULL
);
